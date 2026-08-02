package di

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/samber/do/v2"
	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/config"
	"github.com/whicu/hsa/internal/infrastructure/telemetry"
	"github.com/whicu/hsa/internal/presentation/http"
	"github.com/whicu/hsa/pkg/logger"
)

const (
	defaultConfigPath = "./config/config.yaml"
)

func New() *do.RootScope {
	injector := do.NewWithOpts(&do.InjectorOpts{
		Logf: func(format string, args ...any) {
			fmt.Printf("[DI] "+format+"\n", args...)
		},
		HealthCheckParallelism:   16,
		HealthCheckGlobalTimeout: 20 * time.Second,
	})

	{
		// Inject dependencies
		config.Package(defaultConfigPath)(injector)
		telemetry.Package(injector)
		logger.Package(injector)
		application.Package(injector)
		http.Package(injector)

		// Inject global context
		globalContext(injector)
	}

	return injector
}

func Run(injector *do.RootScope) error {
	_, err := do.Invoke[*telemetry.Service](injector)
	if err != nil {
		return fmt.Errorf("init telemetry service: %w", err)
	}
	log, err := do.Invoke[*slog.Logger](injector)
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}

	// bcfg, err := do.Invoke[broker.Config](injector)
	// if err != nil {
	// 	return fmt.Errorf("init broker config: %w", err)
	// }

	lcfg, err := do.Invoke[logger.Config](injector)
	if err != nil {
		return fmt.Errorf("init logger config: %w", err)
	}
	tcfg, err := do.Invoke[telemetry.Config](injector)
	if err != nil {
		return fmt.Errorf("init service config: %w", err)
	}

	ctx := context.Background()
	log.Log(ctx, 1, "Test", slog.Int("v", 42))
	log.InfoContext(ctx, "Hello Project!", slog.Bool("b", true))
	// log.InfoContext(ctx, "Broker Config", slog.Any("cfg", bcfg))
	log.InfoContext(ctx, "Logger Config", slog.Any("cfg", lcfg))
	log.InfoContext(ctx, "Telemetry Config", slog.Any("cfg", tcfg))

	// srv, err := do.Invoke[micro.Service](injector)
	// if err != nil {
	// 	return fmt.Errorf("init micro service: %w", err)
	// }
	// srv.AddEndpoint(
	// 	"test",
	// 	micro.HandlerFunc(func(r micro.Request) {
	// 		log.Info("Received request", slog.String("subject", r.Subject()))
	// 		log.Info("Received data", slog.String("data", string(r.Data())))
	// 		if err := r.Respond([]byte("Hello from AI Canvas Helper!")); err != nil {
	// 			log.Error("Failed to respond", slog.String("error", err.Error()))
	// 		}
	// 	}),
	// )

	_, report := injector.ShutdownOnSignals()

	if err := report.Error(); err != "" {
		return fmt.Errorf("shutdown: %v", err)
	}
	return nil
}

func globalContext(i do.Injector) {
	ctx := context.Background()
	do.ProvideValue(i, ctx)
}
