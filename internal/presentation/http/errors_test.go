package http

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	api "github.com/whicu/hsa/api/http"
	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/invite"
)

func TestNewVerifyErrors(t *testing.T) {
	reg := newVerifyErrors()
	res, ok := reg.Resolve(ErrUnauthenticated)
	require.True(t, ok)
	require.IsType(t, &api.AuthVerifyGetUnauthorized{}, res)

	res, ok = reg.Resolve(errors.New("unknown"))
	require.True(t, ok)
	require.IsType(t, &api.AuthVerifyGetInternalServerError{}, res)
}

func TestNewInvitesErrors(t *testing.T) {
	reg := newInvitesErrors()
	res, ok := reg.Resolve(ErrUnauthenticated)
	require.True(t, ok)
	require.IsType(t, &api.InvitesPostUnauthorized{}, res)

	res, ok = reg.Resolve(invite.ErrTooManyActive)
	require.True(t, ok)
	require.IsType(t, &api.InvitesPostConflict{}, res)

	res, ok = reg.Resolve(domain.ErrValidation)
	require.True(t, ok)
	require.IsType(t, &api.InvitesPostConflict{}, res)

	res, ok = reg.Resolve(errors.New("unknown"))
	require.True(t, ok)
	require.IsType(t, &api.InvitesPostInternalServerError{}, res)
}

func TestNewRegistrationBeginErrors(t *testing.T) {
	reg := newRegistrationBeginErrors()
	res, ok := reg.Resolve(domain.ErrValidation)
	require.True(t, ok)
	require.IsType(t, &api.RegistrationBeginPostBadRequest{}, res)

	res, ok = reg.Resolve(domain.ErrNotFound)
	require.True(t, ok)
	require.IsType(t, &api.RegistrationBeginPostNotFound{}, res)

	res, ok = reg.Resolve(invite.ErrAlreadyUsed)
	require.True(t, ok)
	require.IsType(t, &api.RegistrationBeginPostConflict{}, res)

	res, ok = reg.Resolve(invite.ErrExpired)
	require.True(t, ok)
	require.IsType(t, &api.RegistrationBeginPostConflict{}, res)

	res, ok = reg.Resolve(errors.New("unknown"))
	require.True(t, ok)
	require.IsType(t, &api.RegistrationBeginPostInternalServerError{}, res)
}

func TestNewRegistrationCompleteErrors(t *testing.T) {
	reg := newRegistrationCompleteErrors()
	res, ok := reg.Resolve(application.ErrWrappedKeysRequired)
	require.True(t, ok)
	require.IsType(t, &api.RegistrationCompletePostBadRequest{}, res)

	res, ok = reg.Resolve(application.ErrInviteNotFound)
	require.True(t, ok)
	require.IsType(t, &api.RegistrationCompletePostNotFound{}, res)

	res, ok = reg.Resolve(invite.ErrAlreadyUsed)
	require.True(t, ok)
	require.IsType(t, &api.RegistrationCompletePostConflict{}, res)

	res, ok = reg.Resolve(invite.ErrExpired)
	require.True(t, ok)
	require.IsType(t, &api.RegistrationCompletePostConflict{}, res)

	res, ok = reg.Resolve(ErrClientIPUnavailable)
	require.True(t, ok)
	require.IsType(t, &api.RegistrationCompletePostInternalServerError{}, res)

	res, ok = reg.Resolve(errors.New("unknown"))
	require.True(t, ok)
	require.IsType(t, &api.RegistrationCompletePostInternalServerError{}, res)
}

func TestNewLoginBeginErrors(t *testing.T) {
	reg := newLoginBeginErrors()
	res, ok := reg.Resolve(errors.New("unknown"))
	require.True(t, ok)
	require.IsType(t, &api.ErrorResponse{}, res)
}

func TestNewLoginErrors(t *testing.T) {
	reg := newLoginErrors()
	res, ok := reg.Resolve(domain.ErrValidation)
	require.True(t, ok)
	require.IsType(t, &api.LoginPostBadRequest{}, res)

	res, ok = reg.Resolve(application.ErrCredentialNotFound)
	require.True(t, ok)
	require.IsType(t, &api.LoginPostUnauthorized{}, res)

	res, ok = reg.Resolve(application.ErrCredentialRevoked)
	require.True(t, ok)
	require.IsType(t, &api.LoginPostUnauthorized{}, res)

	res, ok = reg.Resolve(application.ErrCredentialCloneSuspected)
	require.True(t, ok)
	require.IsType(t, &api.LoginPostUnauthorized{}, res)

	res, ok = reg.Resolve(errors.New("unknown"))
	require.True(t, ok)
	require.IsType(t, &api.LoginPostInternalServerError{}, res)
}

func TestNewRefreshErrors(t *testing.T) {
	reg := newRefreshErrors()
	res, ok := reg.Resolve(application.ErrRefreshTokenNotFound)
	require.True(t, ok)
	require.IsType(t, &api.TokenRefreshPostUnauthorized{}, res)

	res, ok = reg.Resolve(application.ErrRefreshTokenInvalid)
	require.True(t, ok)
	require.IsType(t, &api.TokenRefreshPostUnauthorized{}, res)

	res, ok = reg.Resolve(application.ErrRefreshTokenReuseDetected)
	require.True(t, ok)
	require.IsType(t, &api.TokenRefreshPostUnauthorized{}, res)

	res, ok = reg.Resolve(errors.New("unknown"))
	require.True(t, ok)
	require.IsType(t, &api.TokenRefreshPostInternalServerError{}, res)
}

func TestNewRevokeChainErrors(t *testing.T) {
	reg := newRevokeChainErrors()
	res, ok := reg.Resolve(errors.New("unknown"))
	require.True(t, ok)
	require.IsType(t, &api.AdminUsersUserIdRevokeChainPostInternalServerError{}, res)
}

func TestNewRevokeSessionErrors(t *testing.T) {
	reg := newRevokeSessionErrors()
	res, ok := reg.Resolve(application.ErrSessionNotFound)
	require.True(t, ok)
	require.IsType(t, &api.SessionsSessionIdDeleteNotFound{}, res)

	res, ok = reg.Resolve(application.ErrSessionForbidden)
	require.True(t, ok)
	require.IsType(t, &api.SessionsSessionIdDeleteForbidden{}, res)

	res, ok = reg.Resolve(errors.New("unknown"))
	require.True(t, ok)
	require.IsType(t, &api.SessionsSessionIdDeleteInternalServerError{}, res)
}
