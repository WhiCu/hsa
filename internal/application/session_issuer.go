package application

import (
	"context"
	"log/slog"
	"net/netip"
	"time"

	"github.com/whicu/hsa/internal/domain/session"
	"github.com/whicu/hsa/internal/domain/user"
)

const refreshTokenLength = 32

type RefreshTokenSaver interface {
	Save(ctx context.Context, t ...*session.RefreshToken) error
}

type TokenIssuer interface {
	IssueAccessToken(userID user.UserID, role user.Role, ttl time.Duration) (string, error)
}

type SessionIssuer struct {
	log           *slog.Logger
	sessions      RefreshTokenSaver
	refreshTokens TokenGenerator
	accessTokens  TokenIssuer
	ids           IDGenerator
	users         UserFinderByID
	transactor    Transactor
	refreshTTL    time.Duration
	accessTTL     time.Duration
}

func NewSessionIssuer(
	log *slog.Logger,
	sessions RefreshTokenSaver,
	refreshTokens TokenGenerator,
	accessTokens TokenIssuer,
	ids IDGenerator,
	users UserFinderByID,
	transactor Transactor,
	refreshTTL, accessTTL time.Duration,
) *SessionIssuer {
	return &SessionIssuer{
		log:           log,
		sessions:      sessions,
		refreshTokens: refreshTokens,
		accessTokens:  accessTokens,
		ids:           ids,
		users:         users,
		transactor:    transactor,
		refreshTTL:    refreshTTL,
		accessTTL:     accessTTL,
	}
}

func (si *SessionIssuer) issueAccess(ctx context.Context, userID user.UserID) (string, error) {
	u, err := si.users.FindByID(ctx, userID)
	if err != nil {
		return "", err
	}

	access, err := si.accessTokens.IssueAccessToken(userID, u.Role(), si.accessTTL)
	if err != nil {
		return "", err
	}
	return access, nil
}

func (si *SessionIssuer) Issue(ctx context.Context, userID user.UserID, deviceInfo string, ipAddress netip.Addr, now time.Time) (access, refresh string, err error) {
	si.log.DebugContext(ctx, "issuing session tokens",
		slog.String("user_id", userID.String()),
		slog.String("device_info", deviceInfo),
		slog.String("ip_address", ipAddress.String()),
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

	err = si.transactor.RunInTransaction(ctx, func(txCtx context.Context) error {
		if sessionsErr := si.sessions.Save(txCtx, rt); sessionsErr != nil {
			si.log.ErrorContext(txCtx, "failed to save refresh token session",
				slog.String("user_id", userID.String()),
				slog.String("session_id", rt.ID().String()),
				slog.Any("error", sessionsErr),
			)
			return sessionsErr
		}

		var issueErr error
		access, issueErr = si.issueAccess(txCtx, userID)
		if issueErr != nil {
			si.log.ErrorContext(txCtx, "failed to issue access token",
				slog.String("user_id", userID.String()),
				slog.String("session_id", rt.ID().String()),
				slog.Any("error", issueErr),
			)
			return issueErr
		}

		return nil
	})

	if err != nil {
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

	err = si.transactor.RunInTransaction(ctx, func(txCtx context.Context) error {
		var issueErr error
		access, issueErr = si.issueAccess(txCtx, old.UserID())
		if issueErr != nil {
			si.log.ErrorContext(txCtx, "failed to issue new access token during rotation",
				slog.String("user_id", old.UserID().String()),
				slog.String("old_session_id", old.ID().String()),
				slog.Any("error", issueErr),
			)
			return issueErr
		}

		if sessionsErr := si.sessions.Save(txCtx, old, newRT); sessionsErr != nil {
			si.log.ErrorContext(txCtx, "failed to save rotated session tokens",
				slog.String("user_id", old.UserID().String()),
				slog.String("old_session_id", old.ID().String()),
				slog.String("new_session_id", newRT.ID().String()),
				slog.Any("error", sessionsErr),
			)
			return sessionsErr
		}

		return nil
	})

	if err != nil {
		return "", "", err
	}

	si.log.InfoContext(ctx, "session tokens successfully rotated",
		slog.String("user_id", old.UserID().String()),
		slog.String("old_session_id", old.ID().String()),
		slog.String("new_session_id", newRT.ID().String()),
	)

	return access, rawRefresh, nil
}
