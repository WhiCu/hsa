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
	"github.com/whicu/hsa/internal/domain/invite"
	"github.com/whicu/hsa/internal/domain/user"
)

var ErrChallengeExpired = errors.New("webauthn: challenge expired or invalid")

const prfConst = "prf"

type ChallengeCodec interface {
	Encode(payload any, ttl time.Duration) (token string, err error)
	Decode(token string, out any) error
}

type challengePayload struct {
	SessionData gowebauthn.SessionData `json:"session_data"`
	InviteID    invite.InviteID        `json:"invite_id"`
	UserID      user.UserID            `json:"user_id"`
}

// SECURITY: never log this field
func (p challengePayload) String() string {
	return "challengePayload{SessionData: ***REDACTED***, InviteID: " + p.InviteID.String() + ", UserID: " + p.UserID.String() + "}"
}

type webauthnUser struct {
	id          user.UserID
	credentials []gowebauthn.Credential
}

// SECURITY: never log this field
func (u *webauthnUser) String() string {
	if u == nil {
		return "<nil>"
	}
	return "webauthnUser{id: " + u.id.String() + ", credentials: ***REDACTED***}"
}

func (u *webauthnUser) WebAuthnID() []byte                           { return u.id[:] }
func (u *webauthnUser) WebAuthnName() string                         { return u.id.String() }
func (u *webauthnUser) WebAuthnDisplayName() string                  { return u.id.String() }
func (u *webauthnUser) WebAuthnCredentials() []gowebauthn.Credential { return u.credentials }

type Registrator struct {
	log       *slog.Logger
	wa        *gowebauthn.WebAuthn
	challenge ChallengeCodec
	ttl       time.Duration
}

func NewRegistrator(log *slog.Logger, wa *gowebauthn.WebAuthn, challenge ChallengeCodec, ttl time.Duration) *Registrator {
	return &Registrator{log: log, wa: wa, challenge: challenge, ttl: ttl}
}

func (r *Registrator) Begin(ctx context.Context, candidateUserID user.UserID, inviteID invite.InviteID) (string, []byte, error) {
	r.log.DebugContext(ctx, "beginning webauthn registration",
		slog.String("user_id", candidateUserID.String()),
		slog.String("invite_id", inviteID.String()),
	)

	wu := &webauthnUser{id: candidateUserID}

	options, sessionData, err := r.wa.BeginRegistration(
		wu,
		gowebauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			ResidentKey:      protocol.ResidentKeyRequirementRequired,
			UserVerification: protocol.VerificationRequired,
		}),
		gowebauthn.WithExtensions(protocol.AuthenticationExtensions{
			prfConst: struct{}{},
		}),
	)
	if err != nil {
		r.log.ErrorContext(ctx, "failed to begin webauthn registration",
			slog.String("user_id", candidateUserID.String()),
			slog.String("invite_id", inviteID.String()),
			slog.Any("error", err),
		)
		return "", nil, err
	}

	token, err := r.challenge.Encode(challengePayload{
		SessionData: *sessionData,
		InviteID:    inviteID,
		UserID:      candidateUserID,
	}, r.ttl)
	if err != nil {
		r.log.ErrorContext(ctx, "failed to encode webauthn challenge",
			slog.String("user_id", candidateUserID.String()),
			slog.String("invite_id", inviteID.String()),
			slog.Any("error", err),
		)
		return "", nil, err
	}

	optsJSON, err := json.Marshal(options)
	if err != nil {
		r.log.ErrorContext(ctx, "failed to marshal webauthn options",
			slog.String("user_id", candidateUserID.String()),
			slog.String("invite_id", inviteID.String()),
			slog.Any("error", err),
		)
		return "", nil, err
	}

	r.log.DebugContext(ctx, "webauthn registration options generated successfully",
		slog.String("user_id", candidateUserID.String()),
		slog.String("invite_id", inviteID.String()),
	)

	return token, optsJSON, nil
}

func (r *Registrator) Finish(ctx context.Context, challengeToken string, response []byte) (application.RegistrationResult, error) {
	r.log.DebugContext(ctx, "finishing webauthn registration")

	var payload challengePayload
	if err := r.challenge.Decode(challengeToken, &payload); err != nil {
		r.log.WarnContext(ctx, "failed to decode or validate webauthn challenge token",
			slog.Any("error", err),
		)
		return application.RegistrationResult{}, ErrChallengeExpired
	}

	parsed, err := protocol.ParseCredentialCreationResponseBytes(response)
	if err != nil {
		r.log.ErrorContext(ctx, "failed to parse credential creation response bytes",
			slog.String("user_id", payload.UserID.String()),
			slog.String("invite_id", payload.InviteID.String()),
			slog.Any("error", err),
		)
		return application.RegistrationResult{}, err
	}

	wu := &webauthnUser{id: payload.UserID}
	cred, err := r.wa.CreateCredential(wu, payload.SessionData, parsed)
	if err != nil {
		r.log.ErrorContext(ctx, "failed to create webauthn credential",
			slog.String("user_id", payload.UserID.String()),
			slog.String("invite_id", payload.InviteID.String()),
			slog.Any("error", err),
		)
		return application.RegistrationResult{}, err
	}
	prfSupported := parsePRFExtension(parsed.ClientExtensionResults)

	transports := make([]string, len(cred.Transport))
	for i, t := range cred.Transport {
		transports[i] = string(t)
	}

	r.log.InfoContext(ctx, "webauthn registration successfully finished",
		slog.String("user_id", payload.UserID.String()),
		slog.String("invite_id", payload.InviteID.String()),
		slog.String("credential_id", hex.EncodeToString(cred.ID)),
		slog.Bool("prf_supported", prfSupported),
	)

	return application.RegistrationResult{
		UserID:           payload.UserID,
		InviteID:         payload.InviteID,
		ExternalID:       cred.ID,
		PublicKey:        cred.PublicKey,
		Transports:       transports,
		InitialSignCount: cred.Authenticator.SignCount,
	}, nil
}

func parsePRFExtension(exts protocol.AuthenticationExtensionsClientOutputs) bool {
	prfResult, ok := exts[prfConst]
	if !ok {
		return false
	}
	prfMap, ok := prfResult.(map[string]any)
	if !ok {
		return false
	}
	enabled, ok := prfMap["enabled"].(bool)
	return ok && enabled
}
