package application

import (
	"context"
	"log/slog"
	"time"

	"github.com/whicu/hsa/internal/domain/invite"
	"github.com/whicu/hsa/internal/domain/user"
)

const inviteCodeLength = 32

type InviteSaver interface {
	Save(ctx context.Context, i *invite.Invite) error
}

type ActiveInviteCounter interface {
	CountActiveByUser(ctx context.Context, userID user.UserID, now time.Time) (int, error)
}

type CreateInvite struct {
	log        *slog.Logger
	invites    InviteSaver
	counter    ActiveInviteCounter
	tokens     TokenGenerator
	ids        IDGenerator
	policy     invite.Policy
	transactor Transactor
	ttl        time.Duration
}

func NewCreateInvite(
	log *slog.Logger,
	invites InviteSaver,
	counter ActiveInviteCounter,
	tokens TokenGenerator,
	ids IDGenerator,
	policy invite.Policy,
	transactor Transactor,
	ttl time.Duration,
) *CreateInvite {
	return &CreateInvite{
		log: log, invites: invites, counter: counter, tokens: tokens,
		ids: ids, policy: policy, transactor: transactor, ttl: ttl,
	}
}

func (ci *CreateInvite) Execute(ctx context.Context, createdBy user.UserID) (code string, expiresAt time.Time, err error) {
	ci.log.DebugContext(ctx, "executing create invite",
		slog.String("created_by", createdBy.String()),
	)

	var inv *invite.Invite
	now := time.Now()
	err = ci.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		active, txErr := ci.counter.CountActiveByUser(ctx, createdBy, now) // <- тут совершается pg_advisory_xact_lock
		if txErr != nil {
			ci.log.ErrorContext(ctx, "failed to count active invites for user",
				slog.String("created_by", createdBy.String()), slog.Any("error", txErr))
			return txErr
		}

		if txErr = ci.policy.CanIssue(active); txErr != nil {
			ci.log.WarnContext(ctx, "invite creation rejected by policy",
				slog.String("created_by", createdBy.String()),
				slog.Int("active_invites", active), slog.Any("error", txErr))
			return txErr
		}

		var rawCode, hash string
		rawCode, hash, txErr = ci.tokens.GenerateToken(inviteCodeLength)
		if txErr != nil {
			ci.log.ErrorContext(ctx, "failed to generate invite token",
				slog.String("created_by", createdBy.String()), slog.Any("error", txErr))
			return txErr
		}

		inv, txErr = invite.New(ci.ids.NewID(), createdBy, hash, ci.ttl, now)
		if txErr != nil {
			ci.log.ErrorContext(ctx, "failed to construct invite domain entity",
				slog.String("created_by", createdBy.String()), slog.Any("error", txErr))
			return txErr
		}

		if txErr = ci.invites.Save(ctx, inv); txErr != nil {
			ci.log.ErrorContext(ctx, "failed to save invite",
				slog.String("invite_id", inv.ID().String()),
				slog.String("created_by", createdBy.String()), slog.Any("error", txErr))
			return txErr
		}

		code, expiresAt = rawCode, inv.ExpiresAt()

		ci.log.InfoContext(ctx, "invite successfully created",
			slog.String("invite_id", inv.ID().String()),
			slog.String("created_by", createdBy.String()),
			slog.Time("expires_at", inv.ExpiresAt()))
		return nil
	})
	if err != nil {
		ci.log.ErrorContext(ctx, "failed to create invite", slog.Any("error", err), slog.String("created_by", createdBy.String()))
		return "", time.Time{}, err
	}

	ci.log.InfoContext(ctx, "invite successfully created",
		slog.String("invite_id", inv.ID().String()),
		slog.String("created_by", createdBy.String()),
		slog.Time("expires_at", inv.ExpiresAt()),
	)

	return code, inv.ExpiresAt(), nil
}
