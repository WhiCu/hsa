package application

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/session"
	"github.com/whicu/hsa/internal/domain/user"
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
	sessionSaver  RefreshTokenSaver
	revokeUser    *RevokeAllUserSessions
	accessTokens  TokenIssuer
	refreshTokens TokenGenerator
	hasher        HashGenerator
	ids           IDGenerator
	transactor    Transactor
	refreshTTL    time.Duration
	accessTTL     time.Duration
}

func NewRefreshAccessToken(
	log *slog.Logger,
	sessions RefreshTokenFinder,
	sessionSaver RefreshTokenSaver,
	revokeUser *RevokeAllUserSessions,
	accessTokens TokenIssuer,
	refreshTokens TokenGenerator,
	hasher HashGenerator,
	ids IDGenerator,
	transactor Transactor,
	refreshTTL, accessTTL time.Duration,
) *RefreshAccessToken {
	return &RefreshAccessToken{
		log:           log,
		sessions:      sessions,
		sessionSaver:  sessionSaver,
		revokeUser:    revokeUser,
		accessTokens:  accessTokens,
		refreshTokens: refreshTokens,
		hasher:        hasher,
		ids:           ids,
		transactor:    transactor,
		refreshTTL:    refreshTTL,
		accessTTL:     accessTTL,
	}
}

func (uc *RefreshAccessToken) Execute(ctx context.Context, rawRefreshToken string) (accessToken, refreshToken string, err error) {
	uc.log.DebugContext(ctx, "executing refresh access token")

	now := time.Now()

	oldRT, err := uc.validateRefreshToken(ctx, rawRefreshToken, now)
	if err != nil {
		return "", "", err
	}

	accessToken, refreshToken, newSessionID, err := uc.rotateSession(ctx, oldRT, now)
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
		slog.String("new_session_id", newSessionID.String()),
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

func (uc *RefreshAccessToken) rotateSession(
	ctx context.Context,
	oldRT *session.RefreshToken,
	now time.Time,
) (accessToken, refreshToken string, newSessionID user.UserID, err error) {
	err = uc.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		if txErr := oldRT.Revoke(now); txErr != nil {
			uc.log.ErrorContext(ctx, "failed to revoke old refresh token during rotation",
				slog.String("session_id", oldRT.ID().String()),
				slog.Any("error", txErr),
			)
			return txErr
		}
		if txErr := uc.sessionSaver.Save(ctx, oldRT); txErr != nil {
			uc.log.ErrorContext(ctx, "failed to save revoked old refresh token",
				slog.String("session_id", oldRT.ID().String()),
				slog.Any("error", txErr),
			)
			return txErr
		}

		rawRefresh, refreshHash, txErr := uc.refreshTokens.GenerateToken(refreshTokenLength)
		if txErr != nil {
			uc.log.ErrorContext(ctx, "failed to generate new refresh token string",
				slog.String("user_id", oldRT.UserID().String()),
				slog.Any("error", txErr),
			)
			return txErr
		}

		generatedID := uc.ids.NewID()
		newRT, txErr := session.New(
			generatedID, oldRT.UserID(), refreshHash,
			oldRT.DeviceInfo(), oldRT.IPAddress(),
			uc.refreshTTL, now,
		)
		if txErr != nil {
			uc.log.ErrorContext(ctx, "failed to create new session entity",
				slog.String("user_id", oldRT.UserID().String()),
				slog.Any("error", txErr),
			)
			return txErr
		}

		if ssErr := uc.sessionSaver.Save(ctx, newRT); ssErr != nil {
			uc.log.ErrorContext(ctx, "failed to save new refresh token session",
				slog.String("session_id", newRT.ID().String()),
				slog.Any("error", ssErr),
			)
			return ssErr
		}

		access, txErr := uc.accessTokens.IssueAccessToken(oldRT.UserID(), uc.accessTTL)
		if txErr != nil {
			uc.log.ErrorContext(ctx, "failed to issue new access token",
				slog.String("user_id", oldRT.UserID().String()),
				slog.Any("error", txErr),
			)
			return txErr
		}

		newSessionID = generatedID
		accessToken = access
		refreshToken = rawRefresh
		return nil
	})

	return accessToken, refreshToken, newSessionID, err
}
