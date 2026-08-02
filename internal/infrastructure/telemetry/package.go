package telemetry

import (
	"context"
	"fmt"

	"github.com/knadh/koanf/v2"
	"github.com/samber/do/v2"
	"github.com/whicu/hsa/internal/config"
	"github.com/whicu/hsa/pkg/errkit"
	"go.opentelemetry.io/otel"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
)

type Service struct {
	tp   *sdktrace.TracerProvider
	mp   *sdkmetric.MeterProvider
	lp   *sdklog.LoggerProvider
	conn *grpc.ClientConn
}

func (s *Service) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}
	g := errkit.Group{}

	g.Go(func() error {
		if err := s.tp.Shutdown(ctx); err != nil {
			err = fmt.Errorf("failed to shutdown tracer provider: %w", err)
			otel.Handle(err)
			return err
		}
		return nil
	})

	g.Go(func() error {
		if err := s.mp.Shutdown(ctx); err != nil {
			err = fmt.Errorf("failed to shutdown meter provider: %w", err)
			otel.Handle(err)
			return err
		}
		return nil
	})

	g.Go(func() error {
		if err := s.lp.Shutdown(ctx); err != nil {
			err = fmt.Errorf("failed to shutdown logger provider: %w", err)
			otel.Handle(err)
			return err
		}
		return nil
	})

	g.Finally(func() error {
		if err := s.conn.Close(); err != nil {
			err = fmt.Errorf("failed to close gRPC connection: %w", err)
			otel.Handle(err)
			return err
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return fmt.Errorf("failed to shutdown telemetry: %w", err)
	}
	return nil
}

func newConfig(i do.Injector) (Config, error) {
	k, err := do.Invoke[*koanf.Koanf](i)
	if err != nil {
		return Config{}, err
	}
	def := defaultCfg
	return config.GetConfig(k, "telemetry", &def)
}

func newConn(i do.Injector) (*grpc.ClientConn, error) {
	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, err
	}
	conn, err := InitConn(cfg)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func newResource(i do.Injector) (*resource.Resource, error) {
	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, err
	}
	ctx, err := do.Invoke[context.Context](i)
	if err != nil {
		return nil, err
	}
	return InitResource(ctx, cfg)
}

// func newTracerProvider(i do.Injector) (*sdktrace.TracerProvider, error) {
// 	cfg, err := do.Invoke[Config](i)
// 	if err != nil {
// 		return nil, err
// 	}
// 	ctx, err := do.Invoke[context.Context](i)
// 	if err != nil {
// 		return nil, err
// 	}
// 	res, err := do.Invoke[*resource.Resource](i)
// 	if err != nil {
// 		return nil, err
// 	}
// 	conn, err := do.Invoke[*grpc.ClientConn](i)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return InitTracerProvider(ctx, cfg, res, conn)
// }

// func newMeterProvider(i do.Injector) (*sdkmetric.MeterProvider, error) {
// 	cfg, err := do.Invoke[Config](i)
// 	if err != nil {
// 		return nil, err
// 	}
// 	ctx, err := do.Invoke[context.Context](i)
// 	if err != nil {
// 		return nil, err
// 	}
// 	res, err := do.Invoke[*resource.Resource](i)
// 	if err != nil {
// 		return nil, err
// 	}
// 	conn, err := do.Invoke[*grpc.ClientConn](i)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return InitMeterProvider(ctx, cfg, res, conn)
// }

// func newLoggerProvider(i do.Injector) (*sdklog.LoggerProvider, error) {
// 	cfg, err := do.Invoke[Config](i)
// 	if err != nil {
// 		return nil, err
// 	}
// 	ctx, err := do.Invoke[context.Context](i)
// 	if err != nil {
// 		return nil, err
// 	}
// 	res, err := do.Invoke[*resource.Resource](i)
// 	if err != nil {
// 		return nil, err
// 	}
// 	conn, err := do.Invoke[*grpc.ClientConn](i)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return InitLoggerProvider(ctx, cfg, res, conn)
// }

func newTelemetry(i do.Injector) (*Service, error) {
	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, err
	}
	if !cfg.Enabled {
		//nolint:nilnil // returning nil service when telemetry is disabled is expected by DI container
		return nil, nil
	}

	ctx, err := do.Invoke[context.Context](i)
	if err != nil {
		return nil, err
	}
	res, err := do.Invoke[*resource.Resource](i)
	if err != nil {
		return nil, err
	}
	conn, err := do.Invoke[*grpc.ClientConn](i)
	if err != nil {
		return nil, err
	}
	tp, err := InitTracerProvider(ctx, cfg, res, conn)
	if err != nil {
		return nil, fmt.Errorf("failed to init tracer provider: %w", err)
	}

	mp, err := InitMeterProvider(ctx, cfg, res, conn)
	if err != nil {
		return nil, fmt.Errorf("failed to init meter provider: %w", err)
	}

	lp, err := InitLoggerProvider(ctx, cfg, res, conn)
	if err != nil {
		return nil, fmt.Errorf("failed to init logger provider: %w", err)
	}
	if errRM := InitRuntimeMetrics(); errRM != nil {
		return nil, errRM
	}

	return &Service{tp: tp, mp: mp, lp: lp, conn: conn}, nil
}

var Package = do.Package(
	do.Lazy(newConn),
	do.Lazy(newConfig),
	do.Lazy(newResource),
	// do.Lazy(newTracerProvider),
	// do.Lazy(newMeterProvider),
	// do.Lazy(newLoggerProvider),
	do.Lazy(newTelemetry),
)
