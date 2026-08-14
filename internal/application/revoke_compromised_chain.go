package application

import (
	"context"
	"log/slog"

	"github.com/whicu/hsa/internal/domain/user"
)

type UserDescendantsFinder interface {
	Descendants(ctx context.Context, root user.UserID) ([]user.UserID, error)
}

type RevokeCompromisedChain struct {
	log         *slog.Logger
	descendants UserDescendantsFinder
	revokeUser  *RevokeAllUserSessions
}

func NewRevokeCompromisedChain(
	log *slog.Logger,
	descendants UserDescendantsFinder,
	revokeUser *RevokeAllUserSessions,
) *RevokeCompromisedChain {
	return &RevokeCompromisedChain{
		log:         log,
		descendants: descendants,
		revokeUser:  revokeUser,
	}
}

func (uc *RevokeCompromisedChain) Execute(ctx context.Context, compromisedUserID user.UserID) error {
	uc.log.WarnContext(ctx, "security alert: initiating compromised user chain revocation",
		slog.String("compromised_user_id", compromisedUserID.String()),
	)

	descendantIDs, err := uc.descendants.Descendants(ctx, compromisedUserID)
	if err != nil {
		uc.log.ErrorContext(ctx, "failed to find user descendants for compromised user",
			slog.String("compromised_user_id", compromisedUserID.String()),
			slog.Any("error", err),
		)
		return err
	}

	userIDs := make([]user.UserID, 0, len(descendantIDs))
	userIDs = append(userIDs, descendantIDs...)

	if ruErr := uc.revokeUser.Execute(ctx, userIDs...); ruErr != nil {
		uc.log.ErrorContext(ctx, "failed to revoke sessions for compromised user chain",
			slog.String("compromised_user_id", compromisedUserID.String()),
			slog.Int("descendants_count", len(descendantIDs)),
			slog.Int("total_users_count", len(userIDs)),
			slog.Any("error", ruErr),
		)
		return ruErr
	}

	uc.log.InfoContext(ctx, "successfully revoked sessions for compromised user chain",
		slog.String("compromised_user_id", compromisedUserID.String()),
		slog.Int("descendants_count", len(descendantIDs)),
		slog.Int("total_users_count", len(userIDs)),
	)

	return nil
}
