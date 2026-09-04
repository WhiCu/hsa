package telemetry

import (
	"context"
	"testing"
	"time"

	"github.com/knadh/koanf/v2"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const dummyEndpoint = "localhost:4317"

func TestInitConn(t *testing.T) {
	cfg := defaultCfg
	cfg.Exporter.Conn.Endpoint = dummyEndpoint

	conn, err := InitConn(cfg)
	require.NoError(t, err)
	require.NotNil(t, conn)
	err = conn.Close()
	require.NoError(t, err)

	// Test TLS config
	cfgTLS := cfg
	cfgTLS.Exporter.Conn.Insecure = false
	connTLS, err := InitConn(cfgTLS)
	require.NoError(t, err)
	require.NotNil(t, connTLS)
	err = connTLS.Close()
	require.NoError(t, err)
}

func TestInitResource(t *testing.T) {
	ctx := t.Context()
	cfg := defaultCfg
	res, err := InitResource(ctx, cfg)
	require.NoError(t, err)
	require.NotNil(t, res)
}

func TestInitTracerProvider(t *testing.T) {
	ctx := t.Context()
	cfg := defaultCfg
	res, _ := InitResource(ctx, cfg)

	conn, err := grpc.NewClient(cfg.Exporter.Conn.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	tp, err := InitTracerProvider(ctx, cfg, res, conn)
	require.NoError(t, err)
	require.NotNil(t, tp)

	// Test other samplers
	cfgNever := cfg
	cfgNever.Sampler.Type = "never"
	tpNever, err := InitTracerProvider(ctx, cfgNever, res, conn)
	require.NoError(t, err)
	require.NotNil(t, tpNever)

	cfgRatio := cfg
	cfgRatio.Sampler.Type = "ratio"
	cfgRatio.Sampler.Ratio = 0.5
	tpRatio, err := InitTracerProvider(ctx, cfgRatio, res, conn)
	require.NoError(t, err)
	require.NotNil(t, tpRatio)
}

func TestInitLoggerProvider(t *testing.T) {
	ctx := t.Context()
	cfg := defaultCfg
	res, _ := InitResource(ctx, cfg)

	conn, err := grpc.NewClient(cfg.Exporter.Conn.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	lp, err := InitLoggerProvider(ctx, cfg, res, conn)
	require.NoError(t, err)
	require.NotNil(t, lp)
}

func TestInitMeterProvider(t *testing.T) {
	ctx := t.Context()
	cfg := defaultCfg
	res, _ := InitResource(ctx, cfg)

	conn, err := grpc.NewClient(cfg.Exporter.Conn.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() { conn.Close() })

	mp, err := InitMeterProvider(ctx, cfg, res, conn)
	require.NoError(t, err)
	require.NotNil(t, mp)
}

func TestInitRuntimeMetrics(t *testing.T) {
	err := InitRuntimeMetrics()
	require.NoError(t, err)
}

func TestShutdown(t *testing.T) {
	ctx := t.Context()

	var nilSvc *Service
	err := nilSvc.Shutdown(ctx)
	require.NoError(t, err)
}

func TestNewConfig(t *testing.T) {
	i := do.New()
	do.ProvideValue(i, koanf.New("."))
	cfg, err := newConfig(i)
	require.NoError(t, err)
	require.Equal(t, defaultCfg.ServiceName, cfg.ServiceName)
}

func TestNewConn(t *testing.T) {
	i := do.New()
	cfg := defaultCfg
	cfg.Exporter.Conn.Endpoint = dummyEndpoint
	do.ProvideValue(i, cfg)

	conn, err := newConn(i)
	require.NoError(t, err)
	require.NotNil(t, conn)
	conn.Close()
}

func TestNewResource(t *testing.T) {
	i := do.New()
	do.ProvideValue(i, defaultCfg)
	do.ProvideValue(i, t.Context())

	res, err := newResource(i)
	require.NoError(t, err)
	require.NotNil(t, res)
}

func TestNewTelemetry(t *testing.T) {
	iDisabled := do.New()
	cfgDisabled := defaultCfg
	cfgDisabled.Enabled = false
	do.ProvideValue(iDisabled, cfgDisabled)

	svcDisabled, err := newTelemetry(iDisabled)
	require.NoError(t, err)
	require.Nil(t, svcDisabled)
}

func TestNewTelemetryWithMocks(t *testing.T) {
	i := do.New()
	cfg := defaultCfg
	cfg.Exporter.Conn.Endpoint = dummyEndpoint
	do.ProvideValue(i, cfg)
	do.ProvideValue(i, t.Context())

	res, _ := InitResource(t.Context(), cfg)
	do.ProvideValue(i, res)

	conn, _ := grpc.NewClient(cfg.Exporter.Conn.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	t.Cleanup(func() { conn.Close() })
	do.ProvideValue(i, conn)

	svc, err := newTelemetry(i)
	require.NoError(t, err)
	require.NotNil(t, svc)
}

func TestShutdown_Full(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond*50)
	t.Cleanup(cancel)

	cfg := defaultCfg
	cfg.Exporter.Conn.Endpoint = dummyEndpoint
	res, _ := InitResource(ctx, cfg)

	conn, _ := grpc.NewClient(cfg.Exporter.Conn.Endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))

	tp, _ := InitTracerProvider(ctx, cfg, res, conn)
	lp, _ := InitLoggerProvider(ctx, cfg, res, conn)
	mp, _ := InitMeterProvider(ctx, cfg, res, conn)

	svc := &Service{
		tp:   tp,
		mp:   mp,
		lp:   lp,
		conn: conn,
	}

	err := svc.Shutdown(ctx)
	require.Error(t, err)
}
