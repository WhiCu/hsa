package storage

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/amirsalarsafaei/sqlc-pgx-monitoring/dbtracer"
	"github.com/amirsalarsafaei/sqlc-pgx-monitoring/poolstatus"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/whicu/hsa/internal/infrastructure/storage/migrations"
	"github.com/whicu/hsa/internal/infrastructure/storage/pg"
	"go.opentelemetry.io/otel"
)

const pingTimeout = 5 * time.Second

type Storage struct {
	db      *pgxpool.Pool
	queries *pg.Queries
	log     *slog.Logger
}

func NewStorage(ctx context.Context, log *slog.Logger, cfg Config) (*Storage, error) {
	connCfg, err := pgxpool.ParseConfig(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("parse database config: %w", err)
	}

	connCfg.MaxConns = cfg.MaxOpenConns
	connCfg.MinConns = cfg.MaxIdleConns
	connCfg.MaxConnLifetime = cfg.ConnMaxLifetime
	connCfg.MaxConnIdleTime = cfg.ConnMaxIdleTime

	// telemetry
	{
		mp := otel.GetMeterProvider()
		tp := otel.GetTracerProvider()

		tracer, tracerErr := dbtracer.NewDBTracer(
			cfg.Name,
			dbtracer.WithLogger(log),
			dbtracer.WithMeterProvider(mp),
			dbtracer.WithTraceProvider(tp),
			dbtracer.WithIncludeSpanNameSuffix(true),
			dbtracer.WithLogArgs(false),
		)
		if tracerErr != nil {
			return nil, fmt.Errorf("create db tracer: %w", tracerErr)
		}
		connCfg.ConnConfig.Tracer = tracer
	}

	db, openErr := pgxpool.NewWithConfig(ctx, connCfg)
	if openErr != nil {
		return nil, fmt.Errorf("create database pool: %w", openErr)
	}

	if regErr := poolstatus.Register(db); regErr != nil {
		log.Warn("failed to register pool status monitoring", slog.Any("error", regErr))
	}

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if pingErr := db.Ping(pingCtx); pingErr != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", pingErr)
	}

	queries := pg.New(db)
	return &Storage{
		db:      db,
		queries: queries,
		log:     log,
	}, nil
}

func (s *Storage) Ping(ctx context.Context) error {
	return s.db.Ping(ctx)
}

func (s *Storage) Up(ctx context.Context) error {
	sqlDB := stdlib.OpenDB(*s.db.Config().ConnConfig)
	defer sqlDB.Close()

	if err := migrations.Up(ctx, sqlDB); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	return nil
}

func (s *Storage) Reset(ctx context.Context) error {
	sqlDB := stdlib.OpenDB(*s.db.Config().ConnConfig)
	defer sqlDB.Close()

	if err := migrations.Reset(ctx, sqlDB); err != nil {
		return fmt.Errorf("reset: %w", err)
	}

	return nil
}

func (s *Storage) Shutdown() {
	s.db.Close()
}
