package http

import (
	api "github.com/whicu/hsa/api/http"
	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/invite"
	"github.com/whicu/hsa/pkg/errkit"
)

type IError interface {
	~struct {
		Error api.ErrorResponseError `json:"error"`
	}
}

func NewError(code api.ErrorResponseErrorCode, message string) api.ErrorResponseError {
	return api.ErrorResponseError{
		Code:    code,
		Message: message,
	}
}

func NewErrorResponse[T IError](code api.ErrorResponseErrorCode, message string) *T {
	return &T{
		Error: NewError(code, message),
	}
}

func newVerifyErrors() *errkit.Registry[api.AuthVerifyGetRes] {
	return errkit.New(
		errkit.OnIs(
			func(error) api.AuthVerifyGetRes {
				return NewErrorResponse[api.AuthVerifyGetUnauthorized](
					api.ErrorResponseErrorCodeUNAUTHORIZED,
					"invalid access token or access token expired",
				)
			},
			ErrUnauthenticated,
		),

		errkit.Default(
			func(error) api.AuthVerifyGetRes {
				return NewErrorResponse[api.AuthVerifyGetInternalServerError](
					api.ErrorResponseErrorCodeINTERNALERROR,
					"internal server error",
				)
			},
		),
	)
}

func newInvitesErrors() *errkit.Registry[api.InvitesPostRes] {
	return errkit.New(
		errkit.OnIs(
			func(error) api.InvitesPostRes {
				return NewErrorResponse[api.InvitesPostUnauthorized](
					api.ErrorResponseErrorCodeUNAUTHORIZED,
					"unauthorized",
				)
			},
			ErrUnauthenticated,
		),
		errkit.OnIs(
			func(error) api.InvitesPostRes {
				return NewErrorResponse[api.InvitesPostConflict](
					api.ErrorResponseErrorCodeINVITELIMITEXCEEDED,
					"active invite limit exceeded",
				)
			},
			invite.ErrTooManyActive,
		),
		errkit.OnIs(
			func(error) api.InvitesPostRes {
				return NewErrorResponse[api.InvitesPostConflict](
					api.ErrorResponseErrorCodeVALIDATIONERROR,
					"invalid request",
				)
			},
			domain.ErrValidation,
		),
		errkit.Default(
			func(error) api.InvitesPostRes {
				return NewErrorResponse[api.InvitesPostInternalServerError](
					api.ErrorResponseErrorCodeINTERNALERROR,
					"internal server error",
				)
			},
		),
	)
}

//nolint:dupl // шаблонный маппинг ошибок для сгенерированных типов API
func newRegistrationBeginErrors() *errkit.Registry[api.RegistrationBeginPostRes] {
	return errkit.New(
		errkit.OnIs(
			func(error) api.RegistrationBeginPostRes {
				return NewErrorResponse[api.RegistrationBeginPostBadRequest](
					api.ErrorResponseErrorCodeVALIDATIONERROR,
					"invalid registration request",
				)
			},
			domain.ErrValidation,
		),

		errkit.OnIs(
			func(error) api.RegistrationBeginPostRes {
				return NewErrorResponse[api.RegistrationBeginPostNotFound](
					api.ErrorResponseErrorCodeINVITENOTFOUND,
					"invite not found",
				)
			},
			domain.ErrNotFound,
		),

		errkit.OnIs(
			func(error) api.RegistrationBeginPostRes {
				return NewErrorResponse[api.RegistrationBeginPostConflict](
					api.ErrorResponseErrorCodeINVITEALREADYUSED,
					"invite already used",
				)
			},
			invite.ErrAlreadyUsed,
		),

		errkit.OnIs(
			func(error) api.RegistrationBeginPostRes {
				return NewErrorResponse[api.RegistrationBeginPostConflict](
					api.ErrorResponseErrorCodeINVITEEXPIRED,
					"invite expired",
				)
			},
			invite.ErrExpired,
		),

		errkit.Default(
			func(error) api.RegistrationBeginPostRes {
				return NewErrorResponse[api.RegistrationBeginPostInternalServerError](
					api.ErrorResponseErrorCodeINTERNALERROR,
					"internal server error",
				)
			},
		),
	)
}

func newRegistrationCompleteErrors() *errkit.Registry[api.RegistrationCompletePostRes] {
	return errkit.New(
		errkit.OnIs(
			func(error) api.RegistrationCompletePostRes {
				return NewErrorResponse[api.RegistrationCompletePostBadRequest](
					api.ErrorResponseErrorCodeWRAPPEDKEYSREQUIRED,
					"at least one wrapped key is required",
				)
			},
			application.ErrWrappedKeysRequired,
		),

		errkit.OnIs(
			func(error) api.RegistrationCompletePostRes {
				return NewErrorResponse[api.RegistrationCompletePostNotFound](
					api.ErrorResponseErrorCodeINVITENOTFOUND,
					"invite not found",
				)
			},
			application.ErrInviteNotFound,
		),

		errkit.OnIs(
			func(error) api.RegistrationCompletePostRes {
				return NewErrorResponse[api.RegistrationCompletePostConflict](
					api.ErrorResponseErrorCodeINVITEALREADYUSED,
					"invite already used",
				)
			},
			invite.ErrAlreadyUsed,
		),

		errkit.OnIs(
			func(error) api.RegistrationCompletePostRes {
				return NewErrorResponse[api.RegistrationCompletePostConflict](
					api.ErrorResponseErrorCodeINVITEEXPIRED,
					"invite expired",
				)
			},
			invite.ErrExpired,
		),

		errkit.OnIs(
			func(error) api.RegistrationCompletePostRes {
				return NewErrorResponse[api.RegistrationCompletePostInternalServerError](
					api.ErrorResponseErrorCodeINTERNALERROR,
					"internal server error",
				)
			},
			ErrClientIPUnavailable,
		),

		errkit.Default(
			func(error) api.RegistrationCompletePostRes {
				return NewErrorResponse[api.RegistrationCompletePostInternalServerError](
					api.ErrorResponseErrorCodeINTERNALERROR,
					"internal server error",
				)
			},
		),
	)
}

func newLoginBeginErrors() *errkit.Registry[api.LoginBeginPostRes] {
	return errkit.New(
		errkit.Default(
			func(error) api.LoginBeginPostRes {
				return NewErrorResponse[api.ErrorResponse](
					api.ErrorResponseErrorCodeINTERNALERROR,
					"internal server error",
				)
			},
		),
	)
}

//nolint:dupl // шаблонный маппинг ошибок для сгенерированных типов API
func newLoginErrors() *errkit.Registry[api.LoginPostRes] {
	return errkit.New(
		errkit.OnIs(
			func(error) api.LoginPostRes {
				return NewErrorResponse[api.LoginPostBadRequest](
					api.ErrorResponseErrorCodeVALIDATIONERROR,
					"invalid login request",
				)
			},
			domain.ErrValidation,
		),

		errkit.OnIs(
			func(error) api.LoginPostRes {
				return NewErrorResponse[api.LoginPostUnauthorized](
					api.ErrorResponseErrorCodeCREDENTIALNOTFOUND,
					"credential not found",
				)
			},
			application.ErrCredentialNotFound,
		),

		errkit.OnIs(
			func(error) api.LoginPostRes {
				return NewErrorResponse[api.LoginPostUnauthorized](
					api.ErrorResponseErrorCodeCREDENTIALREVOKED,
					"credential revoked",
				)
			},
			application.ErrCredentialRevoked,
		),

		errkit.OnIs(
			func(error) api.LoginPostRes {
				return NewErrorResponse[api.LoginPostUnauthorized](
					api.ErrorResponseErrorCodeCREDENTIALCLONESUSPECTED,
					"credential clone suspected",
				)
			},
			application.ErrCredentialCloneSuspected,
		),

		errkit.Default(
			func(error) api.LoginPostRes {
				return NewErrorResponse[api.LoginPostInternalServerError](
					api.ErrorResponseErrorCodeINTERNALERROR,
					"internal server error",
				)
			},
		),
	)
}

func newRefreshErrors() *errkit.Registry[api.TokenRefreshPostRes] {
	return errkit.New(
		errkit.OnIs(
			func(error) api.TokenRefreshPostRes {
				return NewErrorResponse[api.TokenRefreshPostUnauthorized](
					api.ErrorResponseErrorCodeREFRESHTOKENNOTFOUND,
					"refresh token not found",
				)
			},
			application.ErrRefreshTokenNotFound,
		),

		errkit.OnIs(
			func(error) api.TokenRefreshPostRes {
				return NewErrorResponse[api.TokenRefreshPostUnauthorized](
					api.ErrorResponseErrorCodeREFRESHTOKENINVALID,
					"refresh token expired or revoked",
				)
			},
			application.ErrRefreshTokenInvalid,
		),

		errkit.OnIs(
			func(error) api.TokenRefreshPostRes {
				return NewErrorResponse[api.TokenRefreshPostUnauthorized](
					api.ErrorResponseErrorCodeREFRESHTOKENREUSEDETECTED,
					"refresh token reuse detected",
				)
			},
			application.ErrRefreshTokenReuseDetected,
		),

		errkit.Default(
			func(error) api.TokenRefreshPostRes {
				return NewErrorResponse[api.TokenRefreshPostInternalServerError](
					api.ErrorResponseErrorCodeINTERNALERROR,
					"internal server error",
				)
			},
		),
	)
}

func newRevokeChainErrors() *errkit.Registry[api.AdminUsersUserIdRevokeChainPostRes] {
	return errkit.New(
		errkit.Default(
			func(error) api.AdminUsersUserIdRevokeChainPostRes {
				return NewErrorResponse[api.AdminUsersUserIdRevokeChainPostInternalServerError](
					api.ErrorResponseErrorCodeINTERNALERROR,
					"internal server error",
				)
			},
		),
	)
}

func newRevokeSessionErrors() *errkit.Registry[api.SessionsSessionIdDeleteRes] {
	return errkit.New(
		errkit.OnIs(
			func(error) api.SessionsSessionIdDeleteRes {
				return NewErrorResponse[api.SessionsSessionIdDeleteNotFound](
					api.ErrorResponseErrorCodeSESSIONNOTFOUND,
					"session not found",
				)
			},
			application.ErrSessionNotFound,
		),

		errkit.OnIs(
			func(error) api.SessionsSessionIdDeleteRes {
				return NewErrorResponse[api.SessionsSessionIdDeleteForbidden](
					api.ErrorResponseErrorCodeSESSIONFORBIDDEN,
					"session belongs to another user",
				)
			},
			application.ErrSessionForbidden,
		),

		errkit.Default(
			func(error) api.SessionsSessionIdDeleteRes {
				return NewErrorResponse[api.SessionsSessionIdDeleteInternalServerError](
					api.ErrorResponseErrorCodeINTERNALERROR,
					"internal server error",
				)
			},
		),
	)
}
