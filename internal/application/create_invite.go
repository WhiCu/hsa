package application

import (
	"context"
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
	invites InviteSaver
	counter ActiveInviteCounter
	tokens  TokenGenerator
	ids     IDGenerator
	policy  invite.Policy
	ttl     time.Duration
}

func NewCreateInvite(invites InviteSaver, counter ActiveInviteCounter, tokens TokenGenerator, ids IDGenerator, policy invite.Policy, ttl time.Duration) *CreateInvite {
	return &CreateInvite{invites: invites, counter: counter, tokens: tokens, ids: ids, policy: policy, ttl: ttl}
}

func (ci *CreateInvite) Execute(ctx context.Context, createdBy user.UserID) (string, time.Time, error) {
	now := time.Now()
	active, err := ci.counter.CountActiveByUser(ctx, createdBy, now)
	if err != nil {
		return "", time.Time{}, err
	}
	if err = ci.policy.CanIssue(active); err != nil {
		return "", time.Time{}, err
	}

	code, hash, err := ci.tokens.GenerateToken(inviteCodeLength)
	if err != nil {
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
		return "", time.Time{}, err
	}

	if err = ci.invites.Save(ctx, inv); err != nil {
		return "", time.Time{}, err
	}
	return code, inv.ExpiresAt(), nil
}
