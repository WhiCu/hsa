package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc"
)

func InitMeterProvider(ctx context.Context, cfg Config, res *resource.Resource, conn *grpc.ClientConn) (*sdkmetric.MeterProvider, error) {
	exporterOpts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithGRPCConn(conn),
		otlpmetricgrpc.WithTimeout(cfg.Exporter.Timeout),
	}

	exporter, err := otlpmetricgrpc.New(ctx, exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP gRPC metric exporter: %w", err)
	}

	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(
			sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(cfg.Metric.Interval)),
		),
	)

	otel.SetMeterProvider(mp)

	return mp, nil
}
