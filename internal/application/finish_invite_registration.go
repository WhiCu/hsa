package application

import (
	"context"
	"errors"
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
	ViaRecovery   bool
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
	now := time.Now()

	result, err := ig.registrator.Finish(ctx, in.ChallengeToken, in.RegistrationResponse)
	if err != nil {
		return nil, err
	}

	var out FinishInviteRegistrationOutput

	err = ig.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		inv, txErr := ig.invites.FindByID(ctx, result.InviteID)
		if txErr != nil {
			if errors.Is(txErr, domain.ErrNotFound) {
				return ErrInviteNotFound
			}
			return txErr
		}
		if txErr = inv.Redeem(result.UserID, now); txErr != nil {
			return txErr
		}
		if txErr = ig.invites.Save(ctx, inv); txErr != nil {
			return txErr
		}

		u, txErr := user.New(result.UserID, inv.CreatedBy(), now)
		if txErr != nil {
			return txErr
		}
		if txErr = ig.users.Save(ctx, u); txErr != nil {
			return txErr
		}

		newCredID := ig.ids.NewID()
		cred, txErr := credential.New(newCredID, result.CredentialID, u.ID(), result.PublicKey, result.Transports, now)
		if txErr != nil {
			return txErr
		}
		if txErr = ig.credentials.Save(ctx, cred); txErr != nil {
			return txErr
		}

		wrapped := make([]*key.WrappedKey, 0, len(in.WrappedKeys))
		for _, wk := range in.WrappedKeys {
			var credID *credential.CredentialID
			if !wk.ViaRecovery {
				cid := cred.ID()
				credID = &cid
			}
			k, kErr := key.New(ig.ids.NewID(), u.ID(), credID, wk.Scope, wk.WrappedDEK, wk.WrapAlgorithm, now)
			if kErr != nil {
				return kErr
			}
			wrapped = append(wrapped, k)
		}
		if txErr = ig.keys.SaveAll(ctx, wrapped); txErr != nil {
			return txErr
		}

		access, refresh, txErr := ig.sessionIssuer.Issue(ctx, u.ID(), in.DeviceInfo, in.IPAddress, now)
		if txErr != nil {
			return txErr
		}

		out = FinishInviteRegistrationOutput{AccessToken: access, RefreshToken: refresh}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}
