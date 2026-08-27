package testutil

import (
	"context"
	"embed"
	"fmt"

	"github.com/moby/moby/api/types/network"
	"github.com/samber/do/v2"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/whicu/hsa/internal/di"
	"github.com/whicu/hsa/internal/infrastructure/storage"
)

//go:embed config/config.yaml
var configFS embed.FS

type Containers struct {
	Postgres *postgres.PostgresContainer
}

func NewTestInjector(ctx context.Context, ctrs Containers) (*do.RootScope, error) {
	injector := di.New(ctx, configFS, "config/config.yaml")
	{
		cfg, err := do.Invoke[storage.Config](injector)
		if err != nil {
			return nil, err
		}

		{
			var host string
			host, err = ctrs.Postgres.Host(ctx)
			if err != nil {
				return nil, fmt.Errorf("get container host: %w", err)
			}
			var port network.Port
			port, err = ctrs.Postgres.MappedPort(ctx, "5432/tcp")
			if err != nil {
				return nil, fmt.Errorf("get mapped port: %w", err)
			}
			cfg.Host = host
			cfg.Port = port.Port()
			cfg.User = TestDBUser
			cfg.Pass = TestDBPass
			cfg.Name = TestDBName
		}
		do.OverrideValue(injector, cfg)
	}

	return injector, nil
}
