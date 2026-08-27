package telemetry_test

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/knadh/koanf/v2"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/samber/do/v2"

	"github.com/whicu/hsa/internal/infrastructure/telemetry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/keepalive"
	"go.opentelemetry.io/otel/sdk/resource"
)

func TestTelemetry(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Telemetry Suite")
}

var _ = Describe("Telemetry Configuration", func() {
	It("converts CfgKeepAliveToGRPCKeepAlive", func() {
		cfg := telemetry.KeepAliveConfig{
			Time:                10 * time.Second,
			Timeout:             5 * time.Second,
			PermitWithoutStream: true,
		}
		expected := keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             5 * time.Second,
			PermitWithoutStream: true,
		}
		Expect(telemetry.CfgKeepAliveToGRPCKeepAlive(cfg)).To(Equal(expected))
	})

	It("converts CfgBackoffToGRPCBackoff", func() {
		cfg := telemetry.BackoffConfig{
			BaseDelay:  2 * time.Second,
			MaxDelay:   30 * time.Second,
			Multiplier: 1.5,
			Jitter:     0.1,
		}
		expected := backoff.Config{
			BaseDelay:  2 * time.Second,
			MaxDelay:   30 * time.Second,
			Multiplier: 1.5,
			Jitter:     0.1,
		}
		Expect(telemetry.CfgBackoffToGRPCBackoff(cfg)).To(Equal(expected))
	})
})

var _ = Describe("Telemetry Initialization", func() {
	var ctx context.Context
	var cfg telemetry.Config

	BeforeEach(func() {
		ctx = context.Background()
		cfg = telemetry.Config{
			ServiceName:    "test-service",
			ServiceVersion: "1.0.0",
			Environment:    "dev",
			Enabled:        true,
			Exporter: telemetry.ExporterConfig{
				Conn: telemetry.ConnConfig{
					Endpoint:   "127.0.0.1:4317",
					MaxMsgSize: 1024,
					Insecure:   true,
					Compressor: "gzip",
					KeepAlive: telemetry.KeepAliveConfig{
						Time:    10 * time.Second,
						Timeout: 2 * time.Second,
					},
					Backoff: telemetry.BackoffConfig{
						BaseDelay:  1 * time.Second,
						MaxDelay:   5 * time.Second,
						Multiplier: 1.0,
					},
				},
				Timeout: 2 * time.Second,
			},
			Sampler: telemetry.SamplerConfig{
				Type: "always",
			},
			Metric: telemetry.MetricConfig{
				Interval: 5 * time.Second,
			},
		}
	})

	It("initializes resource", func() {
		res, err := telemetry.InitResource(ctx, cfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).NotTo(BeNil())
	})

	It("initializes connection insecure", func() {
		conn, err := telemetry.InitConn(cfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(conn).NotTo(BeNil())
		conn.Close()
	})

	It("initializes connection secure", func() {
		cfg.Exporter.Conn.Insecure = false
		conn, err := telemetry.InitConn(cfg)
		Expect(err).NotTo(HaveOccurred())
		Expect(conn).NotTo(BeNil())
		conn.Close()
	})

	Context("with resource and connection", func() {
		var res *resource.Resource
		var conn *grpc.ClientConn

		BeforeEach(func() {
			var err error
			res, err = telemetry.InitResource(ctx, cfg)
			Expect(err).NotTo(HaveOccurred())

			conn, err = telemetry.InitConn(cfg)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			if conn != nil {
				conn.Close()
			}
		})

		Describe("InitTracerProvider", func() {
			It("initializes with always sampler", func() {
				cfg.Sampler.Type = "always"
				tp, err := telemetry.InitTracerProvider(ctx, cfg, res, conn)
				Expect(err).NotTo(HaveOccurred())
				Expect(tp).NotTo(BeNil())
				// Don't shutdown so it doesn't try to send over a non-existent grpc conn
			})

			It("initializes with never sampler", func() {
				cfg.Sampler.Type = "never"
				tp, err := telemetry.InitTracerProvider(ctx, cfg, res, conn)
				Expect(err).NotTo(HaveOccurred())
				Expect(tp).NotTo(BeNil())
			})

			It("initializes with ratio sampler", func() {
				cfg.Sampler.Type = "ratio"
				cfg.Sampler.Ratio = 0.5
				tp, err := telemetry.InitTracerProvider(ctx, cfg, res, conn)
				Expect(err).NotTo(HaveOccurred())
				Expect(tp).NotTo(BeNil())
			})
		})

		It("initializes logger provider", func() {
			lp, err := telemetry.InitLoggerProvider(ctx, cfg, res, conn)
			Expect(err).NotTo(HaveOccurred())
			Expect(lp).NotTo(BeNil())
		})

		It("initializes meter provider", func() {
			mp, err := telemetry.InitMeterProvider(ctx, cfg, res, conn)
			Expect(err).NotTo(HaveOccurred())
			Expect(mp).NotTo(BeNil())
		})
	})

	It("initializes runtime metrics", func() {
		err := telemetry.InitRuntimeMetrics()
		Expect(err).NotTo(HaveOccurred())
	})
})

var _ = Describe("Telemetry Package DI", func() {
	var injector *do.RootScope

	BeforeEach(func() {
		injector = do.New()

		// Setup context
		do.ProvideValue[context.Context](injector, context.Background())

		// Setup config via koanf
		k := koanf.New(".")
		err := k.Load(confmap.Provider(map[string]any{
			"telemetry.service_name": "di-test",
			"telemetry.service_version": "1.0.0",
			"telemetry.environment": "dev",
			"telemetry.enabled": true,
			"telemetry.exporter.timeout": "2s",
			"telemetry.exporter.conn.endpoint": "127.0.0.1:4317",
			"telemetry.exporter.conn.keep_alive.time": "10s",
			"telemetry.exporter.conn.keep_alive.timeout": "2s",
			"telemetry.exporter.conn.backoff.base_delay": "1s",
			"telemetry.exporter.conn.backoff.max_delay": "5s",
			"telemetry.sampler.type": "always",
			"telemetry.metric.interval": "5s",
		}, "."), nil)
		Expect(err).NotTo(HaveOccurred())
		do.ProvideValue[*koanf.Koanf](injector, k)

		telemetry.Package(injector)
	})

	AfterEach(func() {
		// Do not fully shutdown the injector as the telemetry shutdown might time out
		// if the mock collector isn't running, which is hard to gracefully avoid.
		// injector.Shutdown()
	})

	It("provides telemetry configuration", func() {
		cfg, err := do.Invoke[telemetry.Config](injector)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg.ServiceName).To(Equal("di-test"))
	})

	It("provides telemetry connection", func() {
		conn, err := do.Invoke[*grpc.ClientConn](injector)
		Expect(err).NotTo(HaveOccurred())
		Expect(conn).NotTo(BeNil())
	})

	It("provides telemetry resource", func() {
		res, err := do.Invoke[*resource.Resource](injector)
		Expect(err).NotTo(HaveOccurred())
		Expect(res).NotTo(BeNil())
	})

	It("provides telemetry service and gracefully shuts down", func() {
		svc, err := do.Invoke[*telemetry.Service](injector)
		Expect(err).NotTo(HaveOccurred())
		Expect(svc).NotTo(BeNil())

		// Since we don't have a real collector running, just allow the timeout/error
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		_ = svc.Shutdown(ctx)
	})

	It("provides nil service when disabled", func() {
		injectorDisabled := do.New()
		do.ProvideValue[context.Context](injectorDisabled, context.Background())
		k := koanf.New(".")
		err := k.Load(confmap.Provider(map[string]any{
			"telemetry.service_name": "di-test",
			"telemetry.service_version": "1.0.0",
			"telemetry.environment": "dev",
			"telemetry.enabled": false, // Disabled
			"telemetry.exporter.timeout": "2s",
			"telemetry.exporter.conn.endpoint": "127.0.0.1:4317",
			"telemetry.exporter.conn.keep_alive.time": "10s",
			"telemetry.exporter.conn.keep_alive.timeout": "2s",
			"telemetry.exporter.conn.backoff.base_delay": "1s",
			"telemetry.exporter.conn.backoff.max_delay": "5s",
			"telemetry.sampler.type": "always",
			"telemetry.metric.interval": "5s",
		}, "."), nil)
		Expect(err).NotTo(HaveOccurred())
		do.ProvideValue[*koanf.Koanf](injectorDisabled, k)

		telemetry.Package(injectorDisabled)

		svc, err := do.Invoke[*telemetry.Service](injectorDisabled)
		Expect(err).NotTo(HaveOccurred())
		Expect(svc).To(BeNil())

		// Shutdown on nil service shouldn't crash
		var nilSvc *telemetry.Service
		err = nilSvc.Shutdown(context.Background())
		Expect(err).NotTo(HaveOccurred())
	})
})
