package application

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/credential"
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

type Login struct {
	log             *slog.Logger
	credentials     CredentialFinder
	credentialSaver CredentialSaver
	authenticator   Authenticator
	sessionIssuer   *SessionIssuer
	revokeUser      *RevokeAllUserSessions
	transactor      Transactor
}

func NewLogin(
	log *slog.Logger,
	credentials CredentialFinder,
	credentialSaver CredentialSaver,
	authenticator Authenticator,
	sessionIssuer *SessionIssuer,
	revokeUser *RevokeAllUserSessions,
	transactor Transactor,
) *Login {
	return &Login{
		log:             log,
		credentials:     credentials,
		credentialSaver: credentialSaver,
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
	IPAddress              string
}

// SECURITY: never log this field
func (in LoginInput) String() string {
	return fmt.Sprintf("LoginInput{ChallengeToken: ***REDACTED***, AuthenticationResponse: ***REDACTED***, DeviceInfo: %v, IPAddress: %v}", in.DeviceInfo, in.IPAddress)
}

type LoginOutput struct {
	AccessToken  string
	RefreshToken string
}

// SECURITY: never log this field
func (out LoginOutput) String() string {
	return "LoginOutput{AccessToken: ***REDACTED***, RefreshToken: ***REDACTED***}"
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
		out      LoginOutput
		userID   user.UserID
		extIDStr = hex.EncodeToString(result.ExternalID)
	)

	err = uc.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		cred, txErr := uc.credentials.FindByExternalID(ctx, result.ExternalID)
		if txErr != nil {
			if errors.Is(txErr, domain.ErrNotFound) {
				uc.log.WarnContext(ctx, "credential not found during login", slog.String("external_id", extIDStr))
				return ErrCredentialNotFound
			}
			uc.log.ErrorContext(ctx, "failed to find credential by external id", slog.String("external_id", extIDStr), slog.Any("error", txErr))
			return txErr
		}

		if cred.IsRevoked() {
			return ErrCredentialRevoked
		}

		signErr := cred.SetSignCount(result.NewSignCount)
		if result.CloneWarning || errors.Is(signErr, credential.ErrSignCountRegression) {
			uc.log.WarnContext(ctx, "cloned credential detected, revoking key and sessions",
				slog.String("credential_id", cred.ID().String()), slog.String("user_id", cred.UserID().String()), slog.Bool("security", true))

			if credErr := cred.Revoke(now); credErr != nil {
				return credErr
			}
			if credSaveErr := uc.credentialSaver.Save(ctx, cred); credSaveErr != nil {
				return credSaveErr
			}
			if revokeErr := uc.revokeUser.Execute(ctx, cred.UserID()); revokeErr != nil {
				uc.log.ErrorContext(ctx, "failed to revoke user sessions on credential clone detection",
					slog.String("user_id", cred.UserID().String()), slog.Any("error", revokeErr))
				return revokeErr
			}
			return ErrCredentialCloneSuspected
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

		access, refresh, txErr := uc.sessionIssuer.Issue(ctx, cred.UserID(), in.DeviceInfo, in.IPAddress, now)
		if txErr != nil {
			uc.log.ErrorContext(ctx, "failed to issue session tokens during login",
				slog.String("user_id", cred.UserID().String()), slog.Any("error", txErr))
			return txErr
		}

		userID = cred.UserID()
		out = LoginOutput{AccessToken: access, RefreshToken: refresh}
		return nil
	})

	if err != nil {
		uc.log.ErrorContext(ctx, "login transaction failed", slog.String("external_id", extIDStr), slog.Any("error", err))
		return nil, err
	}

	uc.log.InfoContext(ctx, "login successfully finished", slog.String("user_id", userID.String()), slog.String("external_id", extIDStr))

	return &out, nil
}
