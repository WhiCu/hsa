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
	log     *slog.Logger
	invites InviteSaver
	counter ActiveInviteCounter
	tokens  TokenGenerator
	ids     IDGenerator
	policy  invite.Policy
	ttl     time.Duration
}

func NewCreateInvite(
	log *slog.Logger,
	invites InviteSaver,
	counter ActiveInviteCounter,
	tokens TokenGenerator,
	ids IDGenerator,
	policy invite.Policy,
	ttl time.Duration,
) *CreateInvite {
	return &CreateInvite{
		log:     log,
		invites: invites,
		counter: counter,
		tokens:  tokens,
		ids:     ids,
		policy:  policy,
		ttl:     ttl,
	}
}

func (ci *CreateInvite) Execute(ctx context.Context, createdBy user.UserID) (string, time.Time, error) {
	ci.log.DebugContext(ctx, "executing create invite",
		slog.String("created_by", createdBy.String()),
	)

	now := time.Now()
	active, err := ci.counter.CountActiveByUser(ctx, createdBy, now)
	if err != nil {
		ci.log.ErrorContext(ctx, "failed to count active invites for user",
			slog.String("created_by", createdBy.String()),
			slog.Any("error", err),
		)
		return "", time.Time{}, err
	}

	if err = ci.policy.CanIssue(active); err != nil {
		ci.log.WarnContext(ctx, "invite creation rejected by policy",
			slog.String("created_by", createdBy.String()),
			slog.Int("active_invites", active),
			slog.Any("error", err),
		)
		return "", time.Time{}, err
	}

	code, hash, err := ci.tokens.GenerateToken(inviteCodeLength)
	if err != nil {
		ci.log.ErrorContext(ctx, "failed to generate invite token",
			slog.String("created_by", createdBy.String()),
			slog.Any("error", err),
		)
		return "", time.Time{}, err
	}

	inv, err := invite.New(
		ci.ids.NewID(),
		createdBy,
		hash,
		ci.ttl,
		now,
	)
	if err != nil {
		ci.log.ErrorContext(ctx, "failed to construct invite domain entity",
			slog.String("created_by", createdBy.String()),
			slog.Any("error", err),
		)
		return "", time.Time{}, err
	}

	if err = ci.invites.Save(ctx, inv); err != nil {
		ci.log.ErrorContext(ctx, "failed to save invite",
			slog.String("invite_id", inv.ID().String()),
			slog.String("created_by", createdBy.String()),
			slog.Any("error", err),
		)
		return "", time.Time{}, err
	}

	ci.log.InfoContext(ctx, "invite successfully created",
		slog.String("invite_id", inv.ID().String()),
		slog.String("created_by", createdBy.String()),
		slog.Time("expires_at", inv.ExpiresAt()),
	)

	return code, inv.ExpiresAt(), nil
}
