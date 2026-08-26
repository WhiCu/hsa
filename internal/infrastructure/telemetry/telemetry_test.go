package telemetry_test

import (
	"testing"
	"time"

	"github.com/knadh/koanf/v2"
	"github.com/samber/do/v2"
	"github.com/whicu/hsa/internal/infrastructure/telemetry"
)

func TestTelemetryProviders(t *testing.T) {
	ctx := t.Context()
	cfg := telemetry.Config{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		Enabled:        true,
		Exporter: telemetry.ExporterConfig{
			Conn: telemetry.ConnConfig{
				Endpoint: "localhost:4317",
				Insecure: true,
				Compressor: "gzip",
				KeepAlive: telemetry.KeepAliveConfig{
					Time:                10 * time.Second,
					Timeout:             1 * time.Second,
					PermitWithoutStream: true,
				},
				Backoff: telemetry.BackoffConfig{
					BaseDelay:  1 * time.Second,
					MaxDelay:   2 * time.Second,
					Multiplier: 1.5,
					Jitter:     0.1,
				},
			},
			Timeout: 5 * time.Second,
		},
		Sampler: telemetry.SamplerConfig{
			Type: "always",
		},
		Metric: telemetry.MetricConfig{
			Interval: 1 * time.Second,
		},
	}

	res, err := telemetry.InitResource(ctx, cfg)
	if err != nil {
		t.Fatalf("InitResource failed: %v", err)
	}
	if res == nil {
		t.Fatal("resource is nil")
	}

	conn, err := telemetry.InitConn(cfg)
	if err != nil {
		t.Fatalf("InitConn failed: %v", err)
	}
	if conn == nil {
		t.Fatal("conn is nil")
	}
	defer conn.Close()

	tp, err := telemetry.InitTracerProvider(ctx, cfg, res, conn)
	if err != nil {
		t.Fatalf("InitTracerProvider failed: %v", err)
	}
	if tp == nil {
		t.Fatal("tracer provider is nil")
	}
	defer tp.Shutdown(ctx)

	mp, err := telemetry.InitMeterProvider(ctx, cfg, res, conn)
	if err != nil {
		t.Fatalf("InitMeterProvider failed: %v", err)
	}
	if mp == nil {
		t.Fatal("meter provider is nil")
	}
	defer mp.Shutdown(ctx)

	lp, err := telemetry.InitLoggerProvider(ctx, cfg, res, conn)
	if err != nil {
		t.Fatalf("InitLoggerProvider failed: %v", err)
	}
	if lp == nil {
		t.Fatal("logger provider is nil")
	}
	defer lp.Shutdown(ctx)

	err = telemetry.InitRuntimeMetrics()
	if err != nil {
		t.Fatalf("InitRuntimeMetrics failed: %v", err)
	}

	// Test secure conn config
	cfg.Exporter.Conn.Insecure = false
	connSecure, err := telemetry.InitConn(cfg)
	if err != nil {
		t.Fatalf("InitConn (secure) failed: %v", err)
	}
	if connSecure != nil {
		connSecure.Close()
	}

	// Test different samplers
	cfg.Sampler.Type = "never"
	tp2, _ := telemetry.InitTracerProvider(ctx, cfg, res, conn)
	tp2.Shutdown(ctx)

	cfg.Sampler.Type = "ratio"
	cfg.Sampler.Ratio = 0.5
	tp3, _ := telemetry.InitTracerProvider(ctx, cfg, res, conn)
	tp3.Shutdown(ctx)
}

func TestTelemetryPackageDI(t *testing.T) {
	ctx := t.Context()

	i := do.New()
	do.ProvideValue(i, ctx)

	k := koanf.New(".")
	_ = k.Set("telemetry.enabled", true)
	_ = k.Set("telemetry.service_name", "test")
	_ = k.Set("telemetry.service_version", "1.0.0")
	_ = k.Set("telemetry.environment", "dev")
    // Set a very small timeout for the tests to fail faster / avoid blocking timeout errors
    _ = k.Set("telemetry.exporter.timeout", "100ms")
    _ = k.Set("telemetry.metric.interval", "1h")
	do.ProvideValue(i, k)

	telemetry.Package(i)

	svc, err := do.Invoke[*telemetry.Service](i)
	if err != nil {
		t.Fatalf("Failed to invoke telemetry service: %v", err)
	}
	if svc == nil {
		t.Fatal("Expected non-nil service")
	}

	// Shutdown might return an error due to the mock/unreachable collector.
	// That's acceptable as we're just checking that the Shutdown branches are hit.
	_ = svc.Shutdown(ctx)

	var nilSvc *telemetry.Service
	if err := nilSvc.Shutdown(ctx); err != nil {
		t.Fatalf("Expected no error on nil shutdown, got: %v", err)
	}
}

func TestTelemetryPackageDI_Disabled(t *testing.T) {
	ctx := t.Context()
	i := do.New()
	do.ProvideValue(i, ctx)

	k := koanf.New(".")
	_ = k.Set("telemetry.enabled", false)
	_ = k.Set("telemetry.service_name", "test")
	_ = k.Set("telemetry.service_version", "1.0.0")
	_ = k.Set("telemetry.environment", "dev")
	do.ProvideValue(i, k)

	telemetry.Package(i)

	svc, err := do.Invoke[*telemetry.Service](i)
	if err != nil {
		t.Fatalf("Failed to invoke telemetry service: %v", err)
	}
	if svc != nil {
		t.Fatal("Expected nil service since it's disabled")
	}
}
