package telemetry

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
)

func InitTracerProvider(ctx context.Context, cfg Config, res *resource.Resource, conn *grpc.ClientConn) (*sdktrace.TracerProvider, error) {
	exporterOpts := []otlptracegrpc.Option{
		otlptracegrpc.WithGRPCConn(conn),
		otlptracegrpc.WithTimeout(cfg.Exporter.Timeout),
	}

	exporter, err := otlptracegrpc.New(ctx, exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP gRPC trace exporter: %w", err)
	}

	var sampler sdktrace.Sampler
	switch cfg.Sampler.Type {
	case "never":
		sampler = sdktrace.NeverSample()
	case "ratio":
		sampler = sdktrace.TraceIDRatioBased(cfg.Sampler.Ratio)
	default:
		sampler = sdktrace.AlwaysSample()
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sampler)),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}
