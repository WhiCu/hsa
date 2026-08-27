package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/pressly/goose/v3"
)

//go:embed *.sql
var embedMigrations embed.FS

func Up(ctx context.Context, db *sql.DB) error {
	goose.SetBaseFS(embedMigrations)
	goose.SetLogger(goose.NopLogger())

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose set dialect: %w", err)
	}

	if err := goose.UpContext(ctx, db, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}

	return nil
}

func Reset(ctx context.Context, db *sql.DB) error {
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
			AND table_type = 'BASE TABLE'
			AND table_name != 'goose_db_version';
	`

	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("fetch table names for truncate: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if errScan := rows.Scan(&tableName); errScan != nil {
			return fmt.Errorf("scan table name for truncate: %w", errScan)
		}
		tables = append(tables, pgx.Identifier{tableName}.Sanitize())
	}

	if errScan := rows.Err(); errScan != nil {
		return fmt.Errorf("rows iterate error: %w", errScan)
	}

	if len(tables) == 0 {
		return nil
	}

	truncateQuery := fmt.Sprintf("TRUNCATE TABLE %s RESTART IDENTITY CASCADE;", strings.Join(tables, ", "))
	if _, errExec := db.ExecContext(ctx, truncateQuery); errExec != nil {
		return fmt.Errorf("execute dynamic truncate: %w", errExec)
	}

	return nil
}
