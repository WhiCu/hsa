package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/whicu/hsa/internal/domain/session"
	"github.com/whicu/hsa/internal/domain/user"
)

type ActiveSessionsFinder interface {
	FindActiveByUserIDs(ctx context.Context, userIDs []user.UserID, now time.Time) ([]*session.RefreshToken, error)
}

type RefreshTokenBatchSaver interface {
	SaveAll(ctx context.Context, tokens []*session.RefreshToken) error
}

type RevokeAllUserSessions struct {
	log        *slog.Logger
	sessions   ActiveSessionsFinder
	saver      RefreshTokenBatchSaver
	transactor Transactor
}

func NewRevokeAllUserSessions(
	log *slog.Logger,
	sessions ActiveSessionsFinder,
	saver RefreshTokenBatchSaver,
	transactor Transactor,
) *RevokeAllUserSessions {
	return &RevokeAllUserSessions{
		log:        log,
		sessions:   sessions,
		saver:      saver,
		transactor: transactor,
	}
}

func (uc *RevokeAllUserSessions) Execute(ctx context.Context, userIDs ...user.UserID) error {
	// ⚡ Bolt: removed unnecessary userIDStrs allocation; slog natively supports UUID slice serialization
	uc.log.DebugContext(ctx, "executing revoke all user sessions",
		slog.Int("users_count", len(userIDs)),
		slog.Any("user_ids", userIDs),
	)

	now := time.Now()
	var revokedCount int

	err := uc.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		tokens, err := uc.sessions.FindActiveByUserIDs(ctx, userIDs, now)
		if err != nil {
			uc.log.ErrorContext(ctx, "failed to find active sessions for users",
				slog.Any("user_ids", userIDs),
				slog.Any("error", err),
			)
			return err
		}

		for _, t := range tokens {
			if rErr := t.Revoke(now); rErr != nil {
				uc.log.ErrorContext(ctx, "failed to revoke session token",
					slog.String("session_id", t.ID().String()),
					slog.String("user_id", t.UserID().String()),
					slog.Any("error", rErr),
				)
				return rErr
			}
		}

		if sErr := uc.saver.SaveAll(ctx, tokens); sErr != nil {
			uc.log.ErrorContext(ctx, "failed to save revoked sessions",
				slog.Int("tokens_count", len(tokens)),
				slog.Any("error", sErr),
			)
			return sErr
		}

		revokedCount = len(tokens)
		return nil
	})

	if err != nil {
		uc.log.ErrorContext(ctx, "revoke all user sessions transaction failed",
			slog.Any("user_ids", userIDs),
			slog.Any("error", err),
		)
		return err
	}

	uc.log.InfoContext(ctx, "successfully revoked all user sessions",
		slog.Int("users_count", len(userIDs)),
		slog.Int("revoked_sessions_count", revokedCount),
		slog.Any("user_ids", userIDs),
	)

	return nil
}
