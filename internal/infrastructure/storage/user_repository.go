package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/user"
	"github.com/whicu/hsa/internal/infrastructure/storage/pg"
)

type UserRepository struct {
	storage *Storage
}

func NewUserRepository(storage *Storage) *UserRepository {
	return &UserRepository{storage: storage}
}

func (r *UserRepository) Save(ctx context.Context, u *user.User) error {
	return r.storage.GetQueries(ctx).SaveUser(ctx, pg.SaveUserParams{
		ID:        u.ID(),
		InvitedBy: ptrToNullUUID(u.InvitedBy()),
		CreatedAt: u.CreatedAt(),
	})
}

func (r *UserRepository) Descendants(ctx context.Context, root user.UserID) ([]user.UserID, error) {
	ids, err := r.storage.GetQueries(ctx).ListDescendantUserIDs(ctx, root)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if len(ids) == 0 {
		return nil, domain.ErrNotFound
	}
	return ids, nil
}
