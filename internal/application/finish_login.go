package application

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/netip"
	"time"

	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/credential"
	"github.com/whicu/hsa/internal/domain/key"
	"github.com/whicu/hsa/internal/domain/user"
)

var (
	ErrCredentialNotFound       = errors.New("application: credential not found")
	ErrCredentialRevoked        = errors.New("application: credential revoked")
	ErrCredentialCloneSuspected = errors.New("application: credential clone suspected")
)

type CredentialFinder interface {
	FindByExternalID(ctx context.Context, externalID []byte) (*credential.Credential, error)
}

type CredentialWrappedKeysFinder interface {
	FindByCredentialID(ctx context.Context, credentialID credential.CredentialID) ([]*key.WrappedKey, error)
}

type Login struct {
	log             *slog.Logger
	credentials     CredentialFinder
	credentialSaver CredentialSaver
	wrappedKeys     CredentialWrappedKeysFinder
	authenticator   Authenticator
	sessionIssuer   *SessionIssuer
	revokeUser      *RevokeAllUserSessions
	transactor      Transactor
}

func NewLogin(
	log *slog.Logger,
	credentials CredentialFinder,
	credentialSaver CredentialSaver,
	wrappedKeys CredentialWrappedKeysFinder,
	authenticator Authenticator,
	sessionIssuer *SessionIssuer,
	revokeUser *RevokeAllUserSessions,
	transactor Transactor,
) *Login {
	return &Login{
		log:             log,
		credentials:     credentials,
		credentialSaver: credentialSaver,
		wrappedKeys:     wrappedKeys,
		authenticator:   authenticator,
		sessionIssuer:   sessionIssuer,
		revokeUser:      revokeUser,
		transactor:      transactor,
	}
}

type LoginInput struct {
	ChallengeToken         string
	AuthenticationResponse []byte
	DeviceInfo             string
	IPAddress              netip.Addr
}

// SECURITY: never log this field
func (in LoginInput) String() string {
	return fmt.Sprintf("LoginInput{ChallengeToken: ***REDACTED***, AuthenticationResponse: ***REDACTED***, DeviceInfo: %v, IPAddress: %v}", in.DeviceInfo, in.IPAddress)
}

type WrappedKeyOutput struct {
	Scope         key.Scope
	WrappedDEK    []byte
	WrapAlgorithm string
}

func (w WrappedKeyOutput) String() string {
	return fmt.Sprintf("WrappedKeyOutput{Scope: %d, WrappedDEK: %d bytes, WrapAlgorithm: %s}", w.Scope, len(w.WrappedDEK), w.WrapAlgorithm)
}

func wrappedKeysToOutput(keys []*key.WrappedKey) []WrappedKeyOutput {
	out := make([]WrappedKeyOutput, 0, len(keys))
	for _, k := range keys {
		out = append(out, WrappedKeyOutput{
			Scope:         k.Scope(),
			WrappedDEK:    k.WrappedDEK(),
			WrapAlgorithm: k.WrapAlgorithm(),
		})
	}
	return out
}

type LoginOutput struct {
	AccessToken  string
	RefreshToken string
	WrappedKeys  []WrappedKeyOutput
}

// SECURITY: never log this field
func (out LoginOutput) String() string {
	return fmt.Sprintf("LoginOutput{AccessToken: ***REDACTED***, RefreshToken: ***REDACTED***, WrappedKeys: %d records}", len(out.WrappedKeys))
}

func (uc *Login) Execute(ctx context.Context, in LoginInput) (*LoginOutput, error) {
	uc.log.DebugContext(ctx, "executing login")
	now := time.Now()

	result, err := uc.authenticator.Finish(ctx, in.ChallengeToken, in.AuthenticationResponse)
	if err != nil {
		uc.log.ErrorContext(ctx, "failed to finish webauthn login authentication", slog.Any("error", err))
		return nil, err
	}

	var (
		out           LoginOutput
		userID        user.UserID
		extIDStr      = hex.EncodeToString(result.ExternalID)
		cloneDetected bool
	)

	err = uc.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		cred, txErr := uc.findCredential(ctx, result.ExternalID, extIDStr)
		if txErr != nil {
			return txErr
		}

		signErr := cred.SetSignCount(result.NewSignCount)
		if result.CloneWarning || errors.Is(signErr, credential.ErrSignCountRegression) {
			cloneDetected = true
			return uc.handleClone(ctx, cred, now)
		}
		if signErr != nil {
			uc.log.ErrorContext(ctx, "failed to update credential sign count",
				slog.String("credential_id", cred.ID().String()), slog.String("user_id", cred.UserID().String()), slog.Any("error", signErr))
			return signErr
		}

		if csErr := uc.credentialSaver.Save(ctx, cred); csErr != nil {
			uc.log.ErrorContext(ctx, "failed to save updated credential sign count",
				slog.String("credential_id", cred.ID().String()), slog.String("user_id", cred.UserID().String()), slog.Any("error", csErr))
			return csErr
		}

		keys, wkErr := uc.wrappedKeys.FindByCredentialID(ctx, cred.ID())
		if wkErr != nil {
			uc.log.ErrorContext(ctx, "failed to find wrapped keys for credential",
				slog.String("credential_id", cred.ID().String()), slog.Any("error", wkErr))
			return wkErr
		}

		access, refresh, txErr := uc.sessionIssuer.Issue(ctx, cred.UserID(), in.DeviceInfo, in.IPAddress, now)
		if txErr != nil {
			uc.log.ErrorContext(ctx, "failed to issue session tokens during login",
				slog.String("user_id", cred.UserID().String()), slog.Any("error", txErr))
			return txErr
		}

		userID = cred.UserID()
		out = LoginOutput{
			AccessToken:  access,
			RefreshToken: refresh,
			WrappedKeys:  wrappedKeysToOutput(keys),
		}
		return nil
	})

	if err != nil {
		uc.log.ErrorContext(ctx, "login transaction failed", slog.String("external_id", extIDStr), slog.Any("error", err))
		return nil, err
	}

	if cloneDetected {
		return nil, ErrCredentialCloneSuspected
	}

	uc.log.InfoContext(ctx, "login successfully finished", slog.String("user_id", userID.String()), slog.String("external_id", extIDStr), slog.Int("wrapped_keys_count", len(out.WrappedKeys)))
	return &out, nil
}

func (uc *Login) findCredential(ctx context.Context, externalID []byte, extIDStr string) (*credential.Credential, error) {
	cred, err := uc.credentials.FindByExternalID(ctx, externalID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			uc.log.WarnContext(ctx, "credential not found during login", slog.String("external_id", extIDStr))
			return nil, ErrCredentialNotFound
		}
		uc.log.ErrorContext(ctx, "failed to find credential by external id", slog.String("external_id", extIDStr), slog.Any("error", err))
		return nil, err
	}

	if cred.IsRevoked() {
		return nil, ErrCredentialRevoked
	}

	return cred, nil
}

func (uc *Login) handleClone(ctx context.Context, cred *credential.Credential, now time.Time) error {
	uc.log.WarnContext(ctx, "cloned credential detected, revoking credential and sessions",
		slog.String("credential_id", cred.ID().String()),
		slog.String("user_id", cred.UserID().String()),
		slog.Bool("security", true),
	)

	if err := cred.Revoke(now); err != nil {
		return err
	}
	if err := uc.credentialSaver.Save(ctx, cred); err != nil {
		return err
	}
	if err := uc.revokeUser.Execute(ctx, cred.UserID()); err != nil {
		uc.log.ErrorContext(ctx, "failed to revoke user sessions on credential clone detection",
			slog.String("user_id", cred.UserID().String()),
			slog.Any("error", err),
		)
		return err
	}
	return nil
}
