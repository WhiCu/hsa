package http

import (
	"context"
	"log/slog"
	"time"

	api "github.com/whicu/hsa/api/http"
	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/domain/key"
	"github.com/whicu/hsa/internal/domain/session"
	"github.com/whicu/hsa/internal/domain/user"
	"github.com/whicu/hsa/pkg/errkit"
)

type CreateInvite interface {
	Execute(ctx context.Context, createdBy user.UserID) (code string, expiresAt time.Time, err error)
}

type BeginLogin interface {
	Execute(ctx context.Context) (token string, opts []byte, err error)
}

type FinishLogin interface {
	Execute(ctx context.Context, in application.LoginInput) (*application.LoginOutput, error)
}

type BeginInviteRegistration interface {
	Execute(ctx context.Context, inviteCode string) (challengeToken string, creationOptions []byte, err error)
}

type FinishInviteRegistration interface {
	Execute(ctx context.Context, in application.FinishInviteRegistrationInput) (*application.FinishInviteRegistrationOutput, error)
}

type RefreshAccessToken interface {
	Execute(ctx context.Context, rawRefreshToken string) (accessToken string, refreshToken string, err error)
}

type RevokeCompromisedChain interface {
	Execute(ctx context.Context, compromisedUserID user.UserID) error
}

type RevokeSession interface {
	Execute(ctx context.Context, sessionID session.RefreshTokenID, requestingUserID user.UserID) error
}

type Handler struct {
	log           *slog.Logger
	createInvite  CreateInvite
	invitesErrors *errkit.Registry[api.InvitesPostRes]

	beginLogin       BeginLogin
	loginBeginErrors *errkit.Registry[api.LoginBeginPostRes]

	login       FinishLogin
	loginErrors *errkit.Registry[api.LoginPostRes]

	beginRegistration       BeginInviteRegistration
	registrationBeginErrors *errkit.Registry[api.RegistrationBeginPostRes]

	finishRegistration         FinishInviteRegistration
	registrationCompleteErrors *errkit.Registry[api.RegistrationCompletePostRes]

	refreshAccessToken RefreshAccessToken
	refreshErrors      *errkit.Registry[api.TokenRefreshPostRes]

	revokeCompromisedChain RevokeCompromisedChain
	revokeChainErrors      *errkit.Registry[api.AdminUsersUserIdRevokeChainPostRes]

	revokeSession       RevokeSession
	revokeSessionErrors *errkit.Registry[api.SessionsSessionIdDeleteRes]

	verifyErrors *errkit.Registry[api.AuthVerifyGetRes]
}

func NewHandler(
	log *slog.Logger,
	createInvite CreateInvite,
	beginLogin BeginLogin,
	login FinishLogin,
	beginRegistration BeginInviteRegistration,
	finishRegistration FinishInviteRegistration,
	refreshAccessToken RefreshAccessToken,
	revokeCompromisedChain RevokeCompromisedChain,
	revokeSession RevokeSession,
) *Handler {
	return &Handler{
		log:                    log,
		createInvite:           createInvite,
		beginLogin:             beginLogin,
		login:                  login,
		beginRegistration:      beginRegistration,
		finishRegistration:     finishRegistration,
		refreshAccessToken:     refreshAccessToken,
		revokeCompromisedChain: revokeCompromisedChain,
		revokeSession:          revokeSession,

		invitesErrors:              newInvitesErrors(),
		loginBeginErrors:           newLoginBeginErrors(),
		loginErrors:                newLoginErrors(),
		registrationBeginErrors:    newRegistrationBeginErrors(),
		registrationCompleteErrors: newRegistrationCompleteErrors(),
		refreshErrors:              newRefreshErrors(),
		revokeChainErrors:          newRevokeChainErrors(),
		revokeSessionErrors:        newRevokeSessionErrors(),
		verifyErrors:               newVerifyErrors(),
	}
}

var _ api.Handler = (*Handler)(nil)

// AuthVerifyGet implements [api.Handler].
func (h *Handler) AuthVerifyGet(ctx context.Context) (api.AuthVerifyGetRes, error) {
	h.log.DebugContext(ctx, "handling auth verify request")

	userID, ok := userIDFromContext(ctx)
	if !ok {
		h.log.WarnContext(ctx, "auth verify rejected: unauthenticated request")
		return h.verifyErrors.Handle(ErrUnauthenticated)
	}

	h.log.DebugContext(ctx, "auth verify succeeded",
		slog.String("user_id", userID.String()),
	)

	return &api.AuthVerifyGetNoContent{
		XAuthUser: userID,
	}, nil
}

// AdminUsersUserIdRevokeChainPost implements [api.Handler].
//
//nolint:staticcheck // реализует интерфейс, сгенерированный из OpenAPI
func (h *Handler) AdminUsersUserIdRevokeChainPost(
	ctx context.Context,
	params api.AdminUsersUserIdRevokeChainPostParams,
) (api.AdminUsersUserIdRevokeChainPostRes, error) {
	h.log.DebugContext(ctx, "handling revoke compromised chain request",
		slog.String("target_user_id", params.UserId.String()),
	)

	err := h.revokeCompromisedChain.Execute(
		ctx,
		params.UserId,
	)
	if err != nil {
		h.log.ErrorContext(ctx, "failed to revoke compromised chain",
			slog.String("target_user_id", params.UserId.String()),
			slog.Any("error", err),
		)
		return h.revokeChainErrors.Handle(err)
	}

	h.log.InfoContext(ctx, "compromised chain revoked successfully",
		slog.String("target_user_id", params.UserId.String()),
	)

	return &api.AdminUsersUserIdRevokeChainPostNoContent{}, nil
}

// InvitesPost implements [api.Handler].
func (h *Handler) InvitesPost(ctx context.Context) (api.InvitesPostRes, error) {
	h.log.DebugContext(ctx, "handling create invite request")

	userID, ok := userIDFromContext(ctx)
	if !ok {
		h.log.WarnContext(ctx, "create invite rejected: unauthenticated request")
		return h.invitesErrors.Handle(ErrUnauthenticated)
	}

	code, expiresAt, err := h.createInvite.Execute(ctx, userID)
	if err != nil {
		h.log.ErrorContext(ctx, "failed to create invite",
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
		return h.invitesErrors.Handle(err)
	}

	h.log.InfoContext(ctx, "invite created successfully",
		slog.String("user_id", userID.String()),
		slog.Time("expires_at", expiresAt),
	)

	return &api.Invite{
		Code:      code,
		ExpiresAt: expiresAt,
	}, nil
}

// LoginBeginPost implements [api.Handler].
func (h *Handler) LoginBeginPost(
	ctx context.Context,
) (api.LoginBeginPostRes, error) {
	h.log.DebugContext(ctx, "handling login begin request")

	token, opts, err := h.beginLogin.Execute(ctx)
	if err != nil {
		h.log.ErrorContext(ctx, "failed to begin login",
			slog.Any("error", err),
		)
		return h.loginBeginErrors.Handle(err)
	}

	h.log.DebugContext(ctx, "login begin succeeded")

	return &api.LoginBeginPostOK{
		ChallengeToken: token,
		RequestOptions: api.WebAuthnRequestOptions(opts),
	}, nil
}

// LoginPost implements [api.Handler].
func (h *Handler) LoginPost(
	ctx context.Context,
	req *api.LoginPostReq,
) (api.LoginPostRes, error) {
	h.log.DebugContext(ctx, "handling login complete request")

	deviceInfo := req.DeviceInfo.Or("")

	ipAddress, ok := ClientIPFromContext(ctx)
	if !ok {
		h.log.WarnContext(ctx, "login rejected: client ip unavailable")
		return h.loginErrors.Handle(ErrClientIPUnavailable)
	}

	out, err := h.login.Execute(ctx, application.LoginInput{
		ChallengeToken:         req.ChallengeToken,
		AuthenticationResponse: req.GetAuthenticationResponse(),
		DeviceInfo:             deviceInfo,
		IPAddress:              ipAddress,
	})
	if err != nil {
		h.log.ErrorContext(ctx, "failed to complete login",
			slog.Any("error", err),
		)
		return h.loginErrors.Handle(err)
	}

	wrappedKeys := make([]api.WrappedKeyOutput, len(out.WrappedKeys))
	for i, wk := range out.WrappedKeys {
		wrappedKeys[i] = api.WrappedKeyOutput{
			Scope:         api.WrappedKeyScope(key.ScopeToString(wk.Scope)),
			WrappedDek:    wk.WrappedDEK,
			WrapAlgorithm: wk.WrapAlgorithm,
		}
	}

	h.log.InfoContext(ctx, "login request completed successfully",
		slog.Int("wrapped_keys_count", len(wrappedKeys)),
	)

	return &api.LoginPostOK{
		TokenPair: api.TokenPair{
			AccessToken:  out.AccessToken,
			RefreshToken: out.RefreshToken,
		},
		WrappedKeys: wrappedKeys,
	}, nil
}

// RegistrationBeginPost implements [api.Handler].
func (h *Handler) RegistrationBeginPost(
	ctx context.Context,
	req *api.RegistrationBeginPostReq,
) (api.RegistrationBeginPostRes, error) {
	h.log.DebugContext(ctx, "handling registration begin request")

	token, opts, err := h.beginRegistration.Execute(
		ctx,
		req.InviteCode,
	)
	if err != nil {
		h.log.ErrorContext(ctx, "failed to begin registration",
			slog.Any("error", err),
		)
		return h.registrationBeginErrors.Handle(err)
	}

	h.log.DebugContext(ctx, "registration begin succeeded")

	return &api.RegistrationBeginPostOK{
		ChallengeToken:  token,
		CreationOptions: api.WebAuthnCreationOptions(opts),
	}, nil
}

// RegistrationCompletePost implements [api.Handler].
func (h *Handler) RegistrationCompletePost(
	ctx context.Context,
	req *api.RegistrationCompletePostReq,
) (api.RegistrationCompletePostRes, error) {
	h.log.DebugContext(ctx, "handling registration complete request")

	deviceInfo := req.DeviceInfo.Or("")

	ipAddress, ok := ClientIPFromContext(ctx)
	if !ok {
		h.log.WarnContext(ctx, "registration complete rejected: client ip unavailable")
		return h.registrationCompleteErrors.Handle(ErrClientIPUnavailable)
	}

	wrappedKeys := make([]application.WrappedKeyInput, len(req.WrappedKeys))
	for i, wk := range req.WrappedKeys {
		wrappedKeys[i] = application.WrappedKeyInput{
			Scope:         key.ScopeFromString(string(wk.GetScope())),
			WrappedDEK:    wk.WrappedDek,
			WrapAlgorithm: wk.WrapAlgorithm,
		}
	}

	out, err := h.finishRegistration.Execute(
		ctx,
		application.FinishInviteRegistrationInput{
			ChallengeToken:       req.ChallengeToken,
			RegistrationResponse: req.RegistrationResponse,
			WrappedKeys:          wrappedKeys,
			DeviceInfo:           deviceInfo,
			IPAddress:            ipAddress,
		},
	)
	if err != nil {
		h.log.ErrorContext(ctx, "failed to complete registration",
			slog.Any("error", err),
		)
		return h.registrationCompleteErrors.Handle(err)
	}

	h.log.InfoContext(ctx, "registration request completed successfully",
		slog.Int("keys_count", len(wrappedKeys)),
	)

	return &api.TokenPair{
		AccessToken:  out.AccessToken,
		RefreshToken: out.RefreshToken,
	}, nil
}

// SessionsSessionIdDelete implements [api.Handler].
//
//nolint:staticcheck // реализует интерфейс, сгенерированный из OpenAPI
func (h *Handler) SessionsSessionIdDelete(
	ctx context.Context,
	params api.SessionsSessionIdDeleteParams,
) (api.SessionsSessionIdDeleteRes, error) {
	h.log.DebugContext(ctx, "handling revoke session request",
		slog.String("session_id", params.SessionId.String()),
	)

	userID, ok := userIDFromContext(ctx)
	if !ok {
		h.log.WarnContext(ctx, "revoke session rejected: unauthenticated request",
			slog.String("session_id", params.SessionId.String()),
		)
		return h.revokeSessionErrors.Handle(ErrUnauthenticated)
	}

	err := h.revokeSession.Execute(
		ctx,
		params.SessionId,
		userID,
	)
	if err != nil {
		h.log.ErrorContext(ctx, "failed to revoke session",
			slog.String("session_id", params.SessionId.String()),
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
		return h.revokeSessionErrors.Handle(err)
	}

	h.log.InfoContext(ctx, "session revoked successfully",
		slog.String("session_id", params.SessionId.String()),
		slog.String("user_id", userID.String()),
	)

	return &api.SessionsSessionIdDeleteNoContent{}, nil
}

// TokenRefreshPost implements [api.Handler].
func (h *Handler) TokenRefreshPost(
	ctx context.Context,
	req *api.TokenRefreshPostReq,
) (api.TokenRefreshPostRes, error) {
	h.log.DebugContext(ctx, "handling token refresh request")

	access, refresh, err := h.refreshAccessToken.Execute(
		ctx,
		req.RefreshToken,
	)
	if err != nil {
		h.log.ErrorContext(ctx, "failed to refresh access token",
			slog.Any("error", err),
		)
		return h.refreshErrors.Handle(err)
	}

	h.log.InfoContext(ctx, "token refresh completed successfully")

	return &api.TokenPair{
		AccessToken:  access,
		RefreshToken: refresh,
	}, nil
}
