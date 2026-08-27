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
		Role:      pg.UserRole(u.Role().String()), // Добавили сохранение роли
		InvitedBy: ptrToNullUUID(u.InvitedBy()),
		CreatedAt: u.CreatedAt(),
	})
}

// Добавили метод FindByID, который мы теперь используем в SessionIssuer и Login
func (r *UserRepository) FindByID(ctx context.Context, id user.UserID) (*user.User, error) {
	row, err := r.storage.GetQueries(ctx).FindUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return rowToUser(row)
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

func (r *UserRepository) FindRoot(ctx context.Context) (*user.User, error) {
	row, err := r.storage.GetQueries(ctx).FindRootUser(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return rowToUser(row)
}

// Обратите внимание: функция теперь возвращает ошибку,
// так как парсинг роли из строки может завершиться неудачей.
func rowToUser(row pg.User) (*user.User, error) {
	role, err := user.RoleFromString(string(row.Role))
	if err != nil {
		return nil, err
	}

	return user.Reconstruct(
		row.ID,
		role, // Передаем роль в доменную сущность
		nullUUIDToPtr(row.InvitedBy),
		row.CreatedAt,
	), nil
}
