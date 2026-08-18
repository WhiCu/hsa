package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/credential"
	"github.com/whicu/hsa/internal/domain/key"
	"github.com/whicu/hsa/internal/infrastructure/storage/pg"
)

type WrappedKeyRepository struct {
	storage *Storage
}

func NewWrappedKeyRepository(storage *Storage) *WrappedKeyRepository {
	return &WrappedKeyRepository{storage: storage}
}

func (r *WrappedKeyRepository) Save(ctx context.Context, keys ...*key.WrappedKey) error {
	params := make([]pg.SaveWrappedKeysParams, 0, len(keys))
	for _, k := range keys {
		params = append(params, pg.SaveWrappedKeysParams{
			ID:            k.ID(),
			UserID:        k.UserID(),
			CredentialID:  k.CredentialID(),
			Scope:         scopeToPG(k.Scope()),
			WrappedDek:    k.WrappedDEK(),
			WrapAlgorithm: k.WrapAlgorithm(),
			CreatedAt:     k.CreatedAt(),
		})
	}
	_, err := r.storage.GetQueries(ctx).SaveWrappedKeys(ctx, params)
	return err
}

func (r *WrappedKeyRepository) FindByCredentialID(ctx context.Context, credentialID credential.CredentialID) ([]*key.WrappedKey, error) {
	rows, err := r.storage.GetQueries(ctx).ListWrappedKeysByCredentialID(ctx, credentialID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	if len(rows) == 0 {
		return nil, domain.ErrNotFound
	}
	keys := make([]*key.WrappedKey, len(rows))
	for i, row := range rows {
		keys[i] = rowToWrappedKey(row)
	}
	return keys, nil
}

func scopeToPG(s key.Scope) pg.WrappedKeyScope {
	if s == key.ScopeMain {
		return pg.WrappedKeyScopeMain
	}
	return pg.WrappedKeyScopeDecoy
}

func scopeFromPG(s pg.WrappedKeyScope) key.Scope {
	if s == pg.WrappedKeyScopeMain {
		return key.ScopeMain
	}
	return key.ScopeDecoy
}

func rowToWrappedKey(row pg.WrappedKey) *key.WrappedKey {
	return key.Reconstruct(
		row.ID,
		row.UserID,
		row.CredentialID,
		scopeFromPG(row.Scope),
		row.WrappedDek,
		row.WrapAlgorithm,
		row.CreatedAt,
	)
}
