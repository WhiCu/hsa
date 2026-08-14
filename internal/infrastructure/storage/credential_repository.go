package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/credential"
	"github.com/whicu/hsa/internal/domain/user"
	"github.com/whicu/hsa/internal/infrastructure/storage/pg"
)

type CredentialRepository struct {
	storage *Storage
}

func NewCredentialRepository(storage *Storage) *CredentialRepository {
	return &CredentialRepository{storage: storage}
}

func (r *CredentialRepository) FindByExternalID(ctx context.Context, externalID []byte) (*credential.Credential, error) {
	row, err := r.storage.GetQueries(ctx).FindCredentialByExternalID(ctx, externalID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return rowToCredential(row), nil
}

func (r *CredentialRepository) FindByUserID(ctx context.Context, userID user.UserID) ([]*credential.Credential, error) {
	rows, err := r.storage.GetQueries(ctx).ListCredentialsByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	creds := make([]*credential.Credential, 0, len(rows))
	for _, row := range rows {
		creds = append(creds, rowToCredential(row))
	}
	return creds, nil
}

func (r *CredentialRepository) Save(ctx context.Context, c *credential.Credential) error {
	return r.storage.GetQueries(ctx).SaveCredential(ctx, pg.SaveCredentialParams{
		ID:         c.ID(),
		ExternalID: c.ExternalID(),
		UserID:     c.UserID(),
		PublicKey:  c.PublicKey(),
		SignCount:  c.SignCount(),
		Transports: c.Transports(),
		CreatedAt:  c.CreatedAt(),
		RevokedAt:  c.RevokedAt(),
	})
}

func rowToCredential(row pg.Credential) *credential.Credential {
	return credential.Reconstruct(
		row.ID, row.ExternalID, row.UserID, row.PublicKey,
		row.SignCount, row.Transports, row.CreatedAt,
		row.RevokedAt,
	)
}
