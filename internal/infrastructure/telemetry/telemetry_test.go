package telemetry_test

import (
	"context"
	"net"
	"time"

	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"

	"github.com/whicu/hsa/internal/infrastructure/telemetry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var _ = Describe("Telemetry", func() {
	Context("Configuration", func() {
		It("loads defaults when no config provided", func() {
			k := koanf.New(".")
			i := do.New()
			do.ProvideValue(i, k)

			// To test newConfig, since it's lazy provided we can get it via DI
			telemetry.Package(i)

			cfg, err := do.Invoke[telemetry.Config](i)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Enabled).To(BeTrue())
			Expect(cfg.ServiceName).To(Equal("unknown-service"))
		})

		It("loads provided configuration", func() {
			k := koanf.New(".")
			err := k.Load(confmap.Provider(map[string]any{
				"telemetry": map[string]any{
					"enabled": false,
					"service_name": "test-service",
				},
			}, "."), nil)
			Expect(err).NotTo(HaveOccurred())

			i := do.New()
			do.ProvideValue(i, k)
			telemetry.Package(i)

			cfg, err := do.Invoke[telemetry.Config](i)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg.Enabled).To(BeFalse())
			Expect(cfg.ServiceName).To(Equal("test-service"))
		})
	})

	Context("Initialization", func() {
		var ctx context.Context
		var cancel context.CancelFunc

		BeforeEach(func() {
			ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
		})

		AfterEach(func() {
			cancel()
		})

		It("initializes connection with insecure", func() {
			cfg := telemetry.Config{
				Exporter: telemetry.ExporterConfig{
					Conn: telemetry.ConnConfig{
						Endpoint: "localhost:12345",
						Insecure: true,
						MaxMsgSize: 1024,
						Compressor: "gzip",
						KeepAlive: telemetry.KeepAliveConfig{
							Time: 10 * time.Second,
							Timeout: 2 * time.Second,
							PermitWithoutStream: true,
						},
						Backoff: telemetry.BackoffConfig{
							BaseDelay: 1 * time.Second,
							MaxDelay: 5 * time.Second,
							Multiplier: 1.5,
							Jitter: 0.1,
						},
					},
				},
			}

			conn, err := telemetry.InitConn(cfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(conn).NotTo(BeNil())
			defer conn.Close()
		})

		It("initializes connection without insecure (requires actual creds/dial but let's test it builds dialopts)", func() {
			cfg := telemetry.Config{
				Exporter: telemetry.ExporterConfig{
					Conn: telemetry.ConnConfig{
						Endpoint: "localhost:12345",
						Insecure: false, // uses system certs
						Compressor: "",
						KeepAlive: telemetry.KeepAliveConfig{
							Time: 10 * time.Second,
							Timeout: 2 * time.Second,
						},
						Backoff: telemetry.BackoffConfig{
							BaseDelay: 1 * time.Second,
							MaxDelay: 5 * time.Second,
							Multiplier: 1.5,
						},
					},
				},
			}

			conn, err := telemetry.InitConn(cfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(conn).NotTo(BeNil())
			defer conn.Close()
		})

		It("initializes resource", func() {
			cfg := telemetry.Config{
				ServiceName: "test",
				ServiceVersion: "1.0",
				Environment: "test-env",
			}
			res, err := telemetry.InitResource(ctx, cfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(res).NotTo(BeNil())
		})

		Context("Providers", func() {
			var conn *grpc.ClientConn
			var listener net.Listener

			BeforeEach(func() {
				var err error
				listener, err = net.Listen("tcp", "127.0.0.1:0")
				Expect(err).NotTo(HaveOccurred())

				conn, err = grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
				Expect(err).NotTo(HaveOccurred())
			})

			AfterEach(func() {
				conn.Close()
				listener.Close()
			})

			DescribeTable("InitTracerProvider with samplers",
				func(samplerType string) {
					cfg := telemetry.Config{
						Sampler: telemetry.SamplerConfig{
							Type: samplerType,
							Ratio: 0.5,
						},
						Exporter: telemetry.ExporterConfig{
							Timeout: 1 * time.Second,
						},
					}
					res, _ := telemetry.InitResource(ctx, cfg)
					tp, err := telemetry.InitTracerProvider(ctx, cfg, res, conn)
					Expect(err).NotTo(HaveOccurred())
					Expect(tp).NotTo(BeNil())
					tp.Shutdown(ctx)
				},
				Entry("never sampler", "never"),
				Entry("ratio sampler", "ratio"),
				Entry("default/always sampler", "always"),
			)

			It("InitMeterProvider", func() {
				cfg := telemetry.Config{
					Exporter: telemetry.ExporterConfig{
						Timeout: 1 * time.Second,
					},
					Metric: telemetry.MetricConfig{
						Interval: 1 * time.Second,
					},
				}
				res, _ := telemetry.InitResource(ctx, cfg)
				mp, err := telemetry.InitMeterProvider(ctx, cfg, res, conn)
				Expect(err).NotTo(HaveOccurred())
				Expect(mp).NotTo(BeNil())
				mp.Shutdown(ctx)
			})

			It("InitLoggerProvider", func() {
				cfg := telemetry.Config{
					Exporter: telemetry.ExporterConfig{
						Timeout: 1 * time.Second,
					},
				}
				res, _ := telemetry.InitResource(ctx, cfg)
				lp, err := telemetry.InitLoggerProvider(ctx, cfg, res, conn)
				Expect(err).NotTo(HaveOccurred())
				Expect(lp).NotTo(BeNil())
				lp.Shutdown(ctx)
			})
		})
	})

	Context("DI Package Wiring", func() {
		var ctx context.Context
		var cancel context.CancelFunc

		BeforeEach(func() {
			ctx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
		})

		AfterEach(func() {
			cancel()
		})

		It("provides a nil service when disabled", func() {
			k := koanf.New(".")
			k.Load(confmap.Provider(map[string]any{
				"telemetry": map[string]any{
					"enabled": false,
				},
			}, "."), nil)

			i := do.New()
			do.ProvideValue(i, k)
			do.ProvideValue(i, ctx)
			telemetry.Package(i)

			svc, err := do.Invoke[*telemetry.Service](i)
			Expect(err).NotTo(HaveOccurred())
			Expect(svc).To(BeNil())

			err = svc.Shutdown(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("provides a full service when enabled", func() {
			// Spin up a dummy listener to avoid connection refused errors during Shutdown flush
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			Expect(err).NotTo(HaveOccurred())
			defer listener.Close()

			k := koanf.New(".")
			k.Load(confmap.Provider(map[string]any{
				"telemetry": map[string]any{
					"enabled": true,
					"service_name": "test-service",
					"service_version": "1.0.0",
					"environment": "dev",
					"exporter": map[string]any{
						"conn": map[string]any{
							"endpoint": listener.Addr().String(),
							"insecure": true,
							"keep_alive": map[string]any{
								"time": "10s",
								"timeout": "2s",
							},
							"backoff": map[string]any{
								"base_delay": "1s",
								"max_delay": "5s",
							},
						},
						"timeout": "1s",
					},
					"sampler": map[string]any{
						"type": "always",
					},
					"metric": map[string]any{
						"interval": "1s",
					},
				},
			}, "."), nil)

			i := do.New()
			do.ProvideValue(i, k)
			do.ProvideValue(i, ctx)
			telemetry.Package(i)

			svc, err := do.Invoke[*telemetry.Service](i)
			Expect(err).NotTo(HaveOccurred())
			Expect(svc).NotTo(BeNil())

			// Test Shutdown
			// (we expect it might log errors locally or maybe not depending on how graceful shutdown works without a real server, but it shouldn't hang)
			// For testing here, just invoke it to cover the Shutdown method lines.
			_ = svc.Shutdown(ctx)
		})
	})
})
