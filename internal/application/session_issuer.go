package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/whicu/hsa/internal/domain/session"
	"github.com/whicu/hsa/internal/domain/user"
)

const refreshTokenLength = 32

type RefreshTokenSaver interface {
	Save(ctx context.Context, t ...*session.RefreshToken) error
}

type TokenIssuer interface {
	IssueAccessToken(userID user.UserID, ttl time.Duration) (string, error)
}

type SessionIssuer struct {
	log           *slog.Logger
	sessions      RefreshTokenSaver
	refreshTokens TokenGenerator
	accessTokens  TokenIssuer
	ids           IDGenerator
	refreshTTL    time.Duration
	accessTTL     time.Duration
}

func NewSessionIssuer(
	log *slog.Logger,
	sessions RefreshTokenSaver,
	refreshTokens TokenGenerator,
	accessTokens TokenIssuer,
	ids IDGenerator,
	refreshTTL, accessTTL time.Duration,
) *SessionIssuer {
	return &SessionIssuer{
		log:           log,
		sessions:      sessions,
		refreshTokens: refreshTokens,
		accessTokens:  accessTokens,
		ids:           ids,
		refreshTTL:    refreshTTL,
		accessTTL:     accessTTL,
	}
}

func (si *SessionIssuer) Issue(ctx context.Context, userID user.UserID, deviceInfo, ipAddress string, now time.Time) (access, refresh string, err error) {
	si.log.DebugContext(ctx, "issuing session tokens",
		slog.String("user_id", userID.String()),
		slog.String("device_info", deviceInfo),
		slog.String("ip_address", ipAddress),
	)

	rawRefresh, refreshHash, err := si.refreshTokens.GenerateToken(refreshTokenLength)
	if err != nil {
		si.log.ErrorContext(ctx, "failed to generate refresh token string",
			slog.String("user_id", userID.String()),
			slog.Any("error", err),
		)
		return "", "", err
	}

	sessionID := si.ids.NewID()
	rt, err := session.New(sessionID, userID, refreshHash, deviceInfo, ipAddress, si.refreshTTL, now)
	if err != nil {
		si.log.ErrorContext(ctx, "failed to create refresh token session entity",
			slog.String("user_id", userID.String()),
			slog.String("session_id", sessionID.String()),
			slog.Any("error", err),
		)
		return "", "", err
	}

	if sessionsErr := si.sessions.Save(ctx, rt); sessionsErr != nil {
		si.log.ErrorContext(ctx, "failed to save refresh token session",
			slog.String("user_id", userID.String()),
			slog.String("session_id", rt.ID().String()),
			slog.Any("error", sessionsErr),
		)
		return "", "", sessionsErr
	}

	access, err = si.accessTokens.IssueAccessToken(userID, si.accessTTL)
	if err != nil {
		si.log.ErrorContext(ctx, "failed to issue access token",
			slog.String("user_id", userID.String()),
			slog.String("session_id", rt.ID().String()),
			slog.Any("error", err),
		)
		return "", "", err
	}

	si.log.InfoContext(ctx, "session tokens successfully issued",
		slog.String("user_id", userID.String()),
		slog.String("session_id", rt.ID().String()),
	)

	return access, rawRefresh, nil
}

func (si *SessionIssuer) Rotate(ctx context.Context, old *session.RefreshToken, now time.Time) (access, refresh string, err error) {
	si.log.DebugContext(ctx, "rotating session tokens",
		slog.String("user_id", old.UserID().String()),
		slog.String("old_session_id", old.ID().String()),
	)

	rawRefresh, refreshHash, err := si.refreshTokens.GenerateToken(refreshTokenLength)
	if err != nil {
		si.log.ErrorContext(ctx, "failed to generate new refresh token string during rotation",
			slog.String("user_id", old.UserID().String()),
			slog.String("old_session_id", old.ID().String()),
			slog.Any("error", err),
		)
		return "", "", err
	}

	generatedID := si.ids.NewID()
	newRT, err := old.Rotate(generatedID, refreshHash, now)
	if err != nil {
		si.log.ErrorContext(ctx, "failed to rotate session entity",
			slog.String("user_id", old.UserID().String()),
			slog.String("old_session_id", old.ID().String()),
			slog.Any("error", err),
		)
		return "", "", err
	}

	if credErr := old.Revoke(now); credErr != nil {
		si.log.ErrorContext(ctx, "failed to revoke old session token during rotation",
			slog.String("user_id", old.UserID().String()),
			slog.String("old_session_id", old.ID().String()),
			slog.Any("error", credErr),
		)
		return "", "", credErr
	}

	access, err = si.accessTokens.IssueAccessToken(old.UserID(), si.accessTTL)
	if err != nil {
		si.log.ErrorContext(ctx, "failed to issue new access token during rotation",
			slog.String("user_id", old.UserID().String()),
			slog.String("old_session_id", old.ID().String()),
			slog.Any("error", err),
		)
		return "", "", err
	}

	if sessionsErr := si.sessions.Save(ctx, old, newRT); sessionsErr != nil {
		si.log.ErrorContext(ctx, "failed to save rotated session tokens",
			slog.String("user_id", old.UserID().String()),
			slog.String("old_session_id", old.ID().String()),
			slog.String("new_session_id", newRT.ID().String()),
			slog.Any("error", sessionsErr),
		)
		return "", "", sessionsErr
	}

	si.log.InfoContext(ctx, "session tokens successfully rotated",
		slog.String("user_id", old.UserID().String()),
		slog.String("old_session_id", old.ID().String()),
		slog.String("new_session_id", newRT.ID().String()),
	)

	return access, rawRefresh, nil
}
