package application

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/session"
)

var (
	ErrRefreshTokenNotFound      = errors.New("application: refresh token not found")
	ErrRefreshTokenInvalid       = errors.New("application: refresh token expired or revoked")
	ErrRefreshTokenReuseDetected = errors.New("application: refresh token reuse detected")
)

type RefreshTokenFinder interface {
	FindByTokenHash(ctx context.Context, hash string) (*session.RefreshToken, error)
}

type RefreshAccessToken struct {
	log           *slog.Logger
	sessions      RefreshTokenFinder
	revokeUser    *RevokeAllUserSessions
	sessionIssuer *SessionIssuer
	hasher        HashGenerator
}

func NewRefreshAccessToken(
	log *slog.Logger,
	sessions RefreshTokenFinder,
	revokeUser *RevokeAllUserSessions,
	sessionIssuer *SessionIssuer,
	hasher HashGenerator,
) *RefreshAccessToken {
	return &RefreshAccessToken{
		log:           log,
		sessions:      sessions,
		revokeUser:    revokeUser,
		sessionIssuer: sessionIssuer,
		hasher:        hasher,
	}
}

func (uc *RefreshAccessToken) Execute(ctx context.Context, rawRefreshToken string) (accessToken, refreshToken string, err error) {
	uc.log.DebugContext(ctx, "executing refresh access token")

	now := time.Now()

	oldRT, err := uc.validateRefreshToken(ctx, rawRefreshToken, now)
	if err != nil {
		return "", "", err
	}

	accessToken, refreshToken, err = uc.sessionIssuer.Rotate(ctx, oldRT, now)
	if err != nil {
		uc.log.ErrorContext(ctx, "refresh access token transaction failed",
			slog.String("user_id", oldRT.UserID().String()),
			slog.String("old_session_id", oldRT.ID().String()),
			slog.Any("error", err),
		)
		return "", "", err
	}

	uc.log.InfoContext(ctx, "access and refresh tokens successfully rotated",
		slog.String("user_id", oldRT.UserID().String()),
		slog.String("old_session_id", oldRT.ID().String()),
	)

	return accessToken, refreshToken, nil
}

func (uc *RefreshAccessToken) validateRefreshToken(ctx context.Context, rawRefreshToken string, now time.Time) (*session.RefreshToken, error) {
	hash, err := uc.hasher.GenerateHash(rawRefreshToken)
	if err != nil {
		uc.log.ErrorContext(ctx, "failed to generate refresh token hash",
			slog.Any("error", err),
		)
		return nil, err
	}

	oldRT, err := uc.sessions.FindByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			uc.log.WarnContext(ctx, "refresh token not found by hash")
			return nil, ErrRefreshTokenNotFound
		}
		uc.log.ErrorContext(ctx, "failed to find refresh token by hash",
			slog.Any("error", err),
		)
		return nil, err
	}

	if oldRT.IsRevoked() {
		uc.log.WarnContext(ctx, "security alert: refresh token reuse detected, revoking all user sessions",
			slog.String("user_id", oldRT.UserID().String()),
			slog.String("session_id", oldRT.ID().String()),
		)

		if revokeErr := uc.revokeUser.Execute(ctx, oldRT.UserID()); revokeErr != nil {
			uc.log.ErrorContext(ctx, "failed to revoke user sessions upon token reuse detection",
				slog.String("user_id", oldRT.UserID().String()),
				slog.Any("error", revokeErr),
			)
			return nil, revokeErr
		}
		return nil, ErrRefreshTokenReuseDetected
	}

	if !oldRT.IsValid(now) {
		uc.log.WarnContext(ctx, "refresh token is expired or invalid",
			slog.String("user_id", oldRT.UserID().String()),
			slog.String("session_id", oldRT.ID().String()),
		)
		return nil, ErrRefreshTokenInvalid
	}

	return oldRT, nil
}
