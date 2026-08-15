package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/invite"
	"github.com/whicu/hsa/internal/domain/user"
	"github.com/whicu/hsa/internal/infrastructure/storage/pg"
)

type InviteRepository struct {
	storage *Storage
}

func NewInviteRepository(storage *Storage) *InviteRepository {
	return &InviteRepository{storage: storage}
}

func (r *InviteRepository) FindByCodeHash(ctx context.Context, hash string) (*invite.Invite, error) {
	row, err := r.storage.GetQueries(ctx).FindInviteByCodeHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return rowToInvite(row), nil
}

func (r *InviteRepository) FindByID(ctx context.Context, id invite.InviteID) (*invite.Invite, error) {
	row, err := r.storage.GetQueries(ctx).FindInviteByIDForUpdate(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return rowToInvite(row), nil
}

func (r *InviteRepository) Save(ctx context.Context, i *invite.Invite) error {
	return r.storage.GetQueries(ctx).SaveInvite(ctx, pg.SaveInviteParams{
		ID:        i.ID(),
		CreatedBy: i.CreatedBy(),
		CodeHash:  i.CodeHash(),
		UsedBy:    ptrToNullUUID(i.UsedBy()),
		UsedAt:    i.UsedAt(),
		ExpiresAt: i.ExpiresAt(),
		CreatedAt: i.CreatedAt(),
	})
}

func (r *InviteRepository) CountActiveByUser(ctx context.Context, userID user.UserID, now time.Time) (int, error) {
	q := r.storage.GetQueries(ctx)

	if err := q.LockUserByID(ctx, userID.String()); err != nil {
		return 0, err
	}

	count, err := q.CountActiveInvitesByUser(ctx, pg.CountActiveInvitesByUserParams{
		CreatedBy: userID,
		Now:       now,
	})
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

func rowToInvite(row pg.Invite) *invite.Invite {
	return invite.Reconstruct(
		row.ID,
		row.CreatedBy,
		row.CodeHash,
		nullUUIDToPtr(row.UsedBy),
		row.UsedAt,
		row.ExpiresAt,
		row.CreatedAt,
	)
}
