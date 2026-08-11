package storage

import (
	"log/slog"

	"github.com/WhiCu/stgorders/db/pg"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Storage struct {
	db      *pgxpool.Pool
	queries *pg.Queries
	log     *slog.Logger
}
