package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/session"
	"github.com/whicu/hsa/internal/domain/user"
	"github.com/whicu/hsa/internal/infrastructure/storage/pg"
)

type SessionRepository struct {
	storage *Storage
}

func NewSessionRepository(storage *Storage) *SessionRepository {
	return &SessionRepository{storage: storage}
}

func (r *SessionRepository) FindByTokenHash(ctx context.Context, hash string) (*session.RefreshToken, error) {
	row, err := r.storage.GetQueries(ctx).FindRefreshTokenByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return rowToSession(row), nil
}

func (r *SessionRepository) FindByID(ctx context.Context, id session.RefreshTokenID) (*session.RefreshToken, error) {
	row, err := r.storage.GetQueries(ctx).FindRefreshTokenByIDForUpdate(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return rowToSession(row), nil
}

func (r *SessionRepository) FindActiveByUserIDs(ctx context.Context, userIDs []user.UserID, now time.Time) ([]*session.RefreshToken, error) {
	rows, err := r.storage.GetQueries(ctx).ListActiveRefreshTokensByUserIDs(ctx, pg.ListActiveRefreshTokensByUserIDsParams{
		UserIds: userIDs,
		Now:     now,
	})
	if err != nil {
		return nil, err
	}
	tokens := make([]*session.RefreshToken, 0, len(rows))
	for _, row := range rows {
		tokens = append(tokens, rowToSession(row))
	}
	return tokens, nil
}

func (r *SessionRepository) Save(ctx context.Context, tokens ...*session.RefreshToken) error {
	params := make([]pg.SaveRefreshTokensParams, 0, len(tokens))
	for _, t := range tokens {
		params = append(params, pg.SaveRefreshTokensParams{
			ID:         t.ID(),
			UserID:     t.UserID(),
			TokenHash:  t.TokenHash(),
			DeviceInfo: t.DeviceInfo(),
			IpAddress:  t.IPAddress(),
			ExpiresAt:  t.ExpiresAt(),
			RevokedAt:  t.RevokedAt(),
			CreatedAt:  t.CreatedAt(),
		})
	}

	results := r.storage.GetQueries(ctx).SaveRefreshTokens(ctx, params)
	var firstErr error
	results.Exec(func(_ int, err error) {
		if err != nil && firstErr == nil {
			firstErr = err
		}
	})
	if closeErr := results.Close(); closeErr != nil && firstErr == nil {
		firstErr = closeErr
	}
	return firstErr
}

func rowToSession(row pg.RefreshToken) *session.RefreshToken {
	return session.Reconstruct(
		row.ID, row.UserID, row.TokenHash, row.DeviceInfo, row.IpAddress,
		row.ExpiresAt, row.RevokedAt, row.CreatedAt,
	)
}
