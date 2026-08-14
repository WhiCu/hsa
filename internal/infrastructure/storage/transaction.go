package storage

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/whicu/hsa/internal/infrastructure/storage/pg"
	"github.com/whicu/hsa/pkg/errkit"
)

type txKey struct{}

func (s *Storage) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return fn(ctx)
	}

	return s.runInTx(ctx, fn)
}

func (s *Storage) runInTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}

	ctxWithTx := context.WithValue(ctx, txKey{}, tx)

	defer func() {
		if p := recover(); p != nil {
			rollbackErr := tx.Rollback(context.WithoutCancel(ctx))
			if rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
				s.log.ErrorContext(ctx, "storage transaction rollback error on panic", slog.Any("panic", p), slog.Any("error", rollbackErr))
				err = errors.Join(errkit.NewPanicError(p), rollbackErr)
				return
			}
			s.log.ErrorContext(ctx, "storage transaction panic", slog.Any("panic", p))
			err = errkit.NewPanicError(p)
			return
		}
		if err != nil {
			s.log.DebugContext(ctx, "storage transaction rollback", slog.Any("error", err))
			if rollbackErr := tx.Rollback(context.WithoutCancel(ctx)); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
				err = errors.Join(err, rollbackErr)
			}
		}
	}()

	if err = fn(ctxWithTx); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

func (s *Storage) GetQueries(ctx context.Context) *pg.Queries {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return s.queries.WithTx(tx)
	}
	return s.queries
}
