package application

import (
	"context"
	"time"

	"github.com/whicu/hsa/internal/domain/invite"
)

type InviteFinderByCode interface {
	FindByCodeHash(ctx context.Context, hash string) (*invite.Invite, error)
}

type HashGenerator interface {
	GenerateHash(raw string) (string, error)
}

type BeginInviteRegistration struct {
	invites     InviteFinderByCode
	ids         IDGenerator
	registrator Registrator
	hash        HashGenerator
}

func NewBeginInviteRegistration(invites InviteFinderByCode, ids IDGenerator, registrator Registrator, hash HashGenerator) *BeginInviteRegistration {
	return &BeginInviteRegistration{invites: invites, ids: ids, registrator: registrator, hash: hash}
}

func (ig *BeginInviteRegistration) Execute(ctx context.Context, inviteCode string) (challengeToken string, creationOptions []byte, err error) {
	now := time.Now()

	hash, err := ig.hash.GenerateHash(inviteCode)
	if err != nil {
		return "", nil, err
	}

	inv, err := ig.invites.FindByCodeHash(ctx, hash)
	if err != nil {
		return "", nil, err
	}

	if inv.IsUsed() {
		return "", nil, invite.ErrAlreadyUsed
	}
	if inv.IsExpired(now) {
		return "", nil, invite.ErrExpired
	}

	token, opts, err := ig.registrator.Begin(ctx, ig.ids.NewID(), inv.ID())
	if err != nil {
		return "", nil, err
	}
	return token, opts, nil
}
