package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/credential"
	"github.com/whicu/hsa/internal/domain/invite"
	"github.com/whicu/hsa/internal/domain/key"
	"github.com/whicu/hsa/internal/domain/user"
)

var ErrInviteNotFound = errors.New("application: invite not found")

type InviteFinderByID interface {
	FindByID(ctx context.Context, id invite.InviteID) (*invite.Invite, error)
	Save(ctx context.Context, i *invite.Invite) error
}

type UserSaver interface {
	Save(ctx context.Context, u *user.User) error
}

type CredentialSaver interface {
	Save(ctx context.Context, c *credential.Credential) error
}

type WrappedKeySaver interface {
	SaveAll(ctx context.Context, keys []*key.WrappedKey) error
}

type Transactor interface {
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

type FinishInviteRegistration struct {
	log           *slog.Logger
	invites       InviteFinderByID
	users         UserSaver
	credentials   CredentialSaver
	keys          WrappedKeySaver
	sessionIssuer *SessionIssuer
	registrator   Registrator
	ids           IDGenerator
	transactor    Transactor
	refreshTTL    time.Duration
	accessTTL     time.Duration
}

func NewFinishInviteRegistration(
	log *slog.Logger,
	invites InviteFinderByID,
	users UserSaver,
	credentials CredentialSaver,
	keys WrappedKeySaver,
	sessionIssuer *SessionIssuer,
	registrator Registrator,
	ids IDGenerator,
	transactor Transactor,
	refreshTTL time.Duration,
	accessTTL time.Duration,
) *FinishInviteRegistration {
	return &FinishInviteRegistration{
		log:           log,
		invites:       invites,
		users:         users,
		credentials:   credentials,
		keys:          keys,
		sessionIssuer: sessionIssuer,
		registrator:   registrator,
		ids:           ids,
		transactor:    transactor,
		refreshTTL:    refreshTTL,
		accessTTL:     accessTTL,
	}
}

type WrappedKeyInput struct {
	Scope         key.Scope
	WrappedDEK    []byte
	WrapAlgorithm string
}

// SECURITY: never log this field
func (w WrappedKeyInput) String() string {
	return fmt.Sprintf("WrappedKeyInput{Scope: %d, WrappedDEK: ***REDACTED***, WrapAlgorithm: %s}", w.Scope, w.WrapAlgorithm)
}

type FinishInviteRegistrationInput struct {
	ChallengeToken       string
	RegistrationResponse []byte
	WrappedKeys          []WrappedKeyInput
	DeviceInfo           string
	IPAddress            string
}

type FinishInviteRegistrationOutput struct {
	AccessToken  string
	RefreshToken string
}

func (ig *FinishInviteRegistration) Execute(ctx context.Context, in FinishInviteRegistrationInput) (*FinishInviteRegistrationOutput, error) {
	ig.log.DebugContext(ctx, "executing finish invite registration")

	now := time.Now()

	result, err := ig.registrator.Finish(ctx, in.ChallengeToken, in.RegistrationResponse)
	if err != nil {
		ig.log.ErrorContext(ctx, "failed to finish webauthn registration", slog.Any("error", err))
		return nil, err
	}

	var out FinishInviteRegistrationOutput

	err = ig.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		inv, txErr := ig.redeemInvite(ctx, result.InviteID, result.UserID, now)
		if txErr != nil {
			return txErr
		}

		u, cred, txErr := ig.createUserAndCredential(ctx, result, inv.CreatedBy(), now)
		if txErr != nil {
			return txErr
		}

		if txErr = ig.saveWrappedKeys(ctx, u.ID(), cred.ID(), in.WrappedKeys, now); txErr != nil {
			return txErr
		}

		access, refresh, txErr := ig.sessionIssuer.Issue(ctx, u.ID(), in.DeviceInfo, in.IPAddress, now)
		if txErr != nil {
			ig.log.ErrorContext(ctx, "failed to issue session tokens", slog.String("user_id", u.ID().String()), slog.Any("error", txErr))
			return txErr
		}

		out = FinishInviteRegistrationOutput{AccessToken: access, RefreshToken: refresh}
		return nil
	})

	if err != nil {
		ig.log.ErrorContext(ctx, "finish invite registration transaction failed",
			slog.String("user_id", result.UserID.String()),
			slog.String("invite_id", result.InviteID.String()),
			slog.Any("error", err),
		)
		return nil, err
	}

	ig.log.InfoContext(ctx, "invite registration successfully finished",
		slog.String("user_id", result.UserID.String()),
		slog.String("invite_id", result.InviteID.String()),
		slog.Int("keys_count", len(in.WrappedKeys)),
	)

	return &out, nil
}

// --- Internal Transaction Helpers ---

func (ig *FinishInviteRegistration) redeemInvite(ctx context.Context, inviteID invite.InviteID, userID user.UserID, now time.Time) (*invite.Invite, error) {
	inv, err := ig.invites.FindByID(ctx, inviteID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			ig.log.WarnContext(ctx, "invite not found during registration finish", slog.String("invite_id", inviteID.String()))
			return nil, ErrInviteNotFound
		}
		ig.log.ErrorContext(ctx, "failed to find invite by id", slog.String("invite_id", inviteID.String()), slog.Any("error", err))
		return nil, err
	}

	if err = inv.Redeem(userID, now); err != nil {
		ig.log.WarnContext(ctx, "failed to redeem invite", slog.String("invite_id", inv.ID().String()), slog.Any("error", err))
		return nil, err
	}

	if err = ig.invites.Save(ctx, inv); err != nil {
		ig.log.ErrorContext(ctx, "failed to save redeemed invite", slog.String("invite_id", inv.ID().String()), slog.Any("error", err))
		return nil, err
	}

	return inv, nil
}

func (ig *FinishInviteRegistration) createUserAndCredential(ctx context.Context, res RegistrationResult, createdBy user.UserID, now time.Time) (*user.User, *credential.Credential, error) {
	u, err := user.New(res.UserID, createdBy, now)
	if err != nil {
		ig.log.ErrorContext(ctx, "failed to create user domain entity", slog.String("user_id", res.UserID.String()), slog.Any("error", err))
		return nil, nil, err
	}

	if err = ig.users.Save(ctx, u); err != nil {
		ig.log.ErrorContext(ctx, "failed to save user", slog.String("user_id", u.ID().String()), slog.Any("error", err))
		return nil, nil, err
	}

	newCredID := ig.ids.NewID()
	cred, err := credential.New(newCredID, res.CredentialID, u.ID(), res.PublicKey, res.Transports, now)
	if err != nil {
		ig.log.ErrorContext(ctx, "failed to create credential domain entity", slog.String("user_id", u.ID().String()), slog.Any("error", err))
		return nil, nil, err
	}
	if err = cred.SetSignCount(res.InitialSignCount); err != nil {
		ig.log.ErrorContext(ctx, "failed to set initial sign count", slog.String("credential_id", cred.ID().String()), slog.Any("error", err))
		return nil, nil, err
	}

	if err = ig.credentials.Save(ctx, cred); err != nil {
		ig.log.ErrorContext(ctx, "failed to save credential", slog.String("credential_id", cred.ID().String()), slog.Any("error", err))
		return nil, nil, err
	}

	return u, cred, nil
}

func (ig *FinishInviteRegistration) saveWrappedKeys(ctx context.Context, userID user.UserID, credID credential.CredentialID, keyInputs []WrappedKeyInput, now time.Time) error {
	wrapped := make([]*key.WrappedKey, 0, len(keyInputs))
	for _, wk := range keyInputs {
		k, kErr := key.New(ig.ids.NewID(), userID, &credID, wk.Scope, wk.WrappedDEK, wk.WrapAlgorithm, now)
		if kErr != nil {
			ig.log.ErrorContext(ctx, "failed to create wrapped key domain entity", slog.String("user_id", userID.String()), slog.Any("error", kErr))
			return kErr
		}
		wrapped = append(wrapped, k)
	}

	if err := ig.keys.SaveAll(ctx, wrapped); err != nil {
		ig.log.ErrorContext(ctx, "failed to save wrapped keys", slog.String("user_id", userID.String()), slog.Int("keys_count", len(wrapped)), slog.Any("error", err))
		return err
	}

	return nil
}
