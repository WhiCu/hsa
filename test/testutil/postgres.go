package testutil

import (
	"context"
	"fmt"

	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	TestDBName = "test_db"
	TestDBUser = "test_user"
	TestDBPass = "test_password"
)

func StartPostgresContainer(ctx context.Context) (*postgres.PostgresContainer, error) {
	ctr, err := postgres.Run(ctx,
		"postgres:18-alpine",
		postgres.WithDatabase(TestDBName),
		postgres.WithUsername(TestDBUser),
		postgres.WithPassword(TestDBPass),
		postgres.BasicWaitStrategies(),
		postgres.WithSQLDriver("pgx"),
	)
	if err != nil {
		return nil, fmt.Errorf("start postgres container: %w", err)
	}
	return ctr, nil
}
