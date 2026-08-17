package webauthnadapter

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"

	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/domain/credential"
	"github.com/whicu/hsa/internal/domain/user"
)

var ErrUserHandleInvalid = errors.New("webauthn: invalid user handle")

type CredentialsProvider interface {
	FindByUserID(ctx context.Context, userID user.UserID) ([]*credential.Credential, error)
}

type loginChallengePayload struct {
	SessionData gowebauthn.SessionData `json:"session_data"`
}

// SECURITY: never log this field
func (p loginChallengePayload) String() string {
	return "loginChallengePayload{SessionData: ***REDACTED***}"
}

type Authenticator struct {
	log         *slog.Logger
	wa          *gowebauthn.WebAuthn
	challenge   ChallengeCodec
	credentials CredentialsProvider
	ttl         time.Duration
}

func NewAuthenticator(
	log *slog.Logger,
	wa *gowebauthn.WebAuthn,
	challenge ChallengeCodec,
	credentials CredentialsProvider,
	ttl time.Duration,
) *Authenticator {
	return &Authenticator{log: log, wa: wa, challenge: challenge, credentials: credentials, ttl: ttl}
}

func (a *Authenticator) Begin(ctx context.Context) (string, []byte, error) {
	a.log.DebugContext(ctx, "beginning webauthn discoverable login")

	options, sessionData, err := a.wa.BeginDiscoverableLogin(
		gowebauthn.WithUserVerification(protocol.VerificationRequired),
	)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to begin webauthn discoverable login", slog.Any("error", err))
		return "", nil, err
	}

	token, err := a.challenge.Encode(loginChallengePayload{SessionData: *sessionData}, a.ttl)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to encode webauthn login challenge", slog.Any("error", err))
		return "", nil, err
	}

	optsJSON, err := json.Marshal(options)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to marshal webauthn login options", slog.Any("error", err))
		return "", nil, err
	}

	a.log.DebugContext(ctx, "webauthn login options generated successfully")
	return token, optsJSON, nil
}

func (a *Authenticator) Finish(ctx context.Context, challengeToken string, response []byte) (application.AuthenticationResult, error) {
	a.log.DebugContext(ctx, "finishing webauthn login")

	var payload loginChallengePayload
	if err := a.challenge.Decode(challengeToken, &payload); err != nil {
		a.log.WarnContext(ctx, "failed to decode or validate webauthn login challenge token", slog.Any("error", err))
		return application.AuthenticationResult{}, ErrChallengeExpired
	}

	parsed, err := protocol.ParseCredentialRequestResponseBytes(response)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to parse credential request response bytes", slog.Any("error", err))
		return application.AuthenticationResult{}, err
	}

	var userID user.UserID

	_, cred, err := a.wa.ValidatePasskeyLogin(
		a.getValidateFunc(ctx, &userID),
		payload.SessionData,
		parsed,
	)
	if err != nil {
		a.log.ErrorContext(ctx, "failed to validate webauthn discoverable login",
			slog.Any("error", err),
		)
		return application.AuthenticationResult{}, err
	}

	a.log.InfoContext(ctx, "webauthn login successfully finished",
		slog.String("user_id", userID.String()),
		slog.String("credential_id", hex.EncodeToString(cred.ID)),
	)

	return application.AuthenticationResult{
		UserID:       userID,
		ExternalID:   cred.ID,
		NewSignCount: cred.Authenticator.SignCount,
		CloneWarning: cred.Authenticator.CloneWarning,
	}, nil
}

func toWebAuthnCredential(c *credential.Credential) gowebauthn.Credential {
	return gowebauthn.Credential{
		ID:        c.ExternalID(),
		PublicKey: c.PublicKey(),
		Authenticator: gowebauthn.Authenticator{
			SignCount: c.SignCount(),
		},
	}
}

func (a *Authenticator) getValidateFunc(ctx context.Context, userIDOut *user.UserID) func(rawID []byte, userHandle []byte) (gowebauthn.User, error) {
	return func(_, userHandle []byte) (gowebauthn.User, error) {
		candidateID, parseErr := user.NewUserID(userHandle)
		if parseErr != nil {
			a.log.WarnContext(ctx, "invalid user handle in webauthn passkey login",
				slog.Any("error", parseErr),
			)
			return nil, ErrUserHandleInvalid
		}
		*userIDOut = candidateID

		creds, findErr := a.credentials.FindByUserID(ctx, candidateID)
		if findErr != nil {
			a.log.ErrorContext(ctx, "failed to find credentials for user during webauthn login",
				slog.String("user_id", candidateID.String()),
				slog.Any("error", findErr),
			)
			return nil, findErr
		}

		waCreds := make([]gowebauthn.Credential, 0, len(creds))
		for _, c := range creds {
			waCreds = append(waCreds, toWebAuthnCredential(c))
		}

		return &webauthnUser{
			id:          candidateID,
			credentials: waCreds,
		}, nil
	}
}
