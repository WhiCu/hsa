package storage

import (
	"context"

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

func scopeToPG(s key.Scope) pg.WrappedKeyScope {
	if s == key.ScopeDecoy {
		return pg.WrappedKeyScopeDecoy
	}
	return pg.WrappedKeyScopeMain
}
