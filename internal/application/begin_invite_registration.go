package application

import (
	"context"
	"log/slog"
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
	log         *slog.Logger
	invites     InviteFinderByCode
	ids         IDGenerator
	registrator Registrator
	hash        HashGenerator
}

func NewBeginInviteRegistration(
	log *slog.Logger,
	invites InviteFinderByCode,
	ids IDGenerator,
	registrator Registrator,
	hash HashGenerator,
) *BeginInviteRegistration {
	return &BeginInviteRegistration{
		log:         log,
		invites:     invites,
		ids:         ids,
		registrator: registrator,
		hash:        hash,
	}
}

func (ig *BeginInviteRegistration) Execute(ctx context.Context, inviteCode string) (challengeToken string, creationOptions []byte, err error) {
	ig.log.DebugContext(ctx, "executing begin invite registration")

	now := time.Now()

	hash, err := ig.hash.GenerateHash(inviteCode)
	if err != nil {
		ig.log.ErrorContext(ctx, "failed to generate hash for invite code",
			slog.Any("error", err),
		)
		return "", nil, err
	}

	inv, err := ig.invites.FindByCodeHash(ctx, hash)
	if err != nil {
		ig.log.ErrorContext(ctx, "failed to find invite by code hash",
			slog.Any("error", err),
		)
		return "", nil, err
	}

	if inv.IsUsed() {
		ig.log.WarnContext(ctx, "invite registration rejected: invite already used",
			slog.String("invite_id", inv.ID().String()),
		)
		return "", nil, invite.ErrAlreadyUsed
	}

	if inv.IsExpired(now) {
		ig.log.WarnContext(ctx, "invite registration rejected: invite expired",
			slog.String("invite_id", inv.ID().String()),
			slog.Time("expires_at", inv.ExpiresAt()),
		)
		return "", nil, invite.ErrExpired
	}

	candidateUserID := ig.ids.NewID()

	token, opts, err := ig.registrator.Begin(ctx, candidateUserID, inv.ID())
	if err != nil {
		ig.log.ErrorContext(ctx, "failed to begin webauthn registration for invite",
			slog.String("invite_id", inv.ID().String()),
			slog.String("candidate_user_id", candidateUserID.String()),
			slog.Any("error", err),
		)
		return "", nil, err
	}

	ig.log.InfoContext(ctx, "begin invite registration completed successfully",
		slog.String("invite_id", inv.ID().String()),
		slog.String("candidate_user_id", candidateUserID.String()),
	)

	return token, opts, nil
}
