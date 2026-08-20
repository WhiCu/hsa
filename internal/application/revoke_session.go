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
	ErrSessionNotFound  = errors.New("application: session not found")
	ErrSessionForbidden = errors.New("application: session belongs to another user")
)

type RefreshTokenFinderByID interface {
	FindByID(ctx context.Context, id session.RefreshTokenID) (*session.RefreshToken, error)
}

type RevokeSession struct {
	log        *slog.Logger
	sessions   RefreshTokenFinderByID
	saver      RefreshTokenSaver
	transactor Transactor
}

func NewRevokeSession(log *slog.Logger, sessions RefreshTokenFinderByID, saver RefreshTokenSaver, transactor Transactor) *RevokeSession {
	return &RevokeSession{log: log, sessions: sessions, saver: saver, transactor: transactor}
}

func (uc *RevokeSession) Execute(ctx context.Context, sessionID session.RefreshTokenID, requestingUserID user.UserID) error {
	uc.log.DebugContext(ctx, "executing revoke session",
		slog.String("session_id", sessionID.String()),
		slog.String("requesting_user_id", requestingUserID.String()),
	)

	now := time.Now()
	err := uc.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		rt, txErr := uc.sessions.FindByID(ctx, sessionID)
		if txErr != nil {
			if errors.Is(txErr, domain.ErrNotFound) {
				uc.log.WarnContext(ctx, "revoke session: not found", slog.String("session_id", sessionID.String()))
				return ErrSessionNotFound
			}
			uc.log.ErrorContext(ctx, "failed to find session by id",
				slog.String("session_id", sessionID.String()), slog.Any("error", txErr))
			return txErr
		}

		if rt.UserID() != requestingUserID {
			uc.log.WarnContext(ctx, "revoke session: ownership mismatch",
				slog.String("session_id", sessionID.String()),
				slog.String("session_owner_id", rt.UserID().String()),
				slog.String("requesting_user_id", requestingUserID.String()),
			)
			return ErrSessionForbidden
		}

		if errRevoke := rt.Revoke(now); errRevoke != nil {
			if errors.Is(errRevoke, session.ErrAlreadyRevoked) {
				uc.log.DebugContext(ctx, "revoke session: already revoked, no-op",
					slog.String("session_id", sessionID.String()))
				return nil
			}
			uc.log.ErrorContext(ctx, "failed to revoke session",
				slog.String("session_id", sessionID.String()), slog.Any("error", errRevoke))
			return errRevoke
		}

		if errSave := uc.saver.Save(ctx, rt); errSave != nil {
			uc.log.ErrorContext(ctx, "failed to save revoked session",
				slog.String("session_id", sessionID.String()), slog.Any("error", errSave))
			return errSave
		}
		return nil
	})

	if err != nil {
		uc.log.ErrorContext(ctx, "revoke user sessions transaction failed",
			slog.String("session_id", sessionID.String()),
			slog.String("user_id", requestingUserID.String()),
			slog.Any("error", err),
		)
		return err
	}

	uc.log.InfoContext(ctx, "session revoked",
		slog.String("session_id", sessionID.String()),
		slog.String("user_id", requestingUserID.String()),
	)
	return nil
}
