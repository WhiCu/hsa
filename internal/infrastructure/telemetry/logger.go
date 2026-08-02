package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploggrpc"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc"
)

func InitLoggerProvider(ctx context.Context, cfg Config, res *resource.Resource, conn *grpc.ClientConn) (*sdklog.LoggerProvider, error) {
	exporterOpts := []otlploggrpc.Option{
		otlploggrpc.WithGRPCConn(conn),
		otlploggrpc.WithTimeout(cfg.Exporter.Timeout),
	}

	exporter, err := otlploggrpc.New(ctx, exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP gRPC log exporter: %w", err)
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
	)

	global.SetLoggerProvider(lp)

	return lp, nil
}
