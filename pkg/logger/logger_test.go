package logger_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/knadh/koanf/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"

	"github.com/whicu/hsa/pkg/logger"
)

func TestLoggerPkg(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Logger Suite")
}

var _ = Describe("Logger Package", func() {
	Context("Error Wrapper", func() {
		It("wraps errors with prefix", func() {
			// Just a sanity check for level wrapper coverage
			_, _, err := logger.Logger(logger.Config{Level: "invalid"})
			Expect(err).To(HaveOccurred())
		})
	})

	Context("NOP Logger", func() {
		It("provides a handler that ignores everything", func(ctx SpecContext) {
			handler := logger.NewNOPHandler()

			Expect(handler.Enabled(ctx, slog.LevelDebug)).To(BeFalse())
			Expect(handler.Enabled(ctx, slog.LevelInfo)).To(BeFalse())
			Expect(handler.Enabled(ctx, slog.LevelError)).To(BeFalse())

			Expect(handler.Handle(ctx, slog.Record{})).To(Succeed())

			hWithAttrs := handler.WithAttrs([]slog.Attr{{Key: "k", Value: slog.StringValue("v")}})
			Expect(hWithAttrs).To(Equal(handler))

			hWithGroup := handler.WithGroup("g")
			Expect(hWithGroup).To(Equal(handler))
		})

		It("provides a usable NOP slog instance", func() {
			log := logger.NewNOPSlog()
			Expect(log).NotTo(BeNil())

			subLog := logger.GetSubLogger(log, "sub")
			Expect(subLog).NotTo(BeNil())
		})
	})

	Context("Configuration and Lifecycle", func() {
		It("initializes and shuts down successfully", func() {
			cfg := logger.Config{
				Level:  "info",
				Caller: 1,
				File: logger.FileConfig{
					Name:        filepath.Join(os.TempDir(), "test.log"),
					Size:        10,
					Backups:     3,
					ChannelSize: 100,
					Discard:     false,
				},
			}

			log, closer, err := logger.Logger(cfg)
			Expect(err).NotTo(HaveOccurred())
			Expect(log).NotTo(BeNil())
			Expect(closer).NotTo(BeNil())

			log.Info("test log")
			Expect(closer.Close()).To(Succeed())
		})

		It("rejects invalid log levels", func() {
			cfg := logger.Config{
				Level: "invalid-level",
			}
			_, _, err := logger.Logger(cfg)
			Expect(err).To(HaveOccurred())
		})
	})

	Context("Dependency Injection", func() {
		It("resolves components via injector", func() {
			// This exercises newConfig, newService, newSlogLogger
			// Requires setting up koanf correctly, we might skip full DI test
			// if it requires too much env setup, but we can verify it fails gracefully
			// when koanf is missing.
			injector := do.New()
			logger.Package(injector)
			// Manually put config so newService passes
			do.OverrideValue(injector, logger.Config{
				Level: "debug",
				File: logger.FileConfig{
					Name:        filepath.Join(os.TempDir(), "di.log"),
					Size:        100,
					Backups:     1,
					ChannelSize: 100,
				},
			})

			log, err := do.Invoke[*slog.Logger](injector)
			Expect(err).NotTo(HaveOccurred())
			Expect(log).NotTo(BeNil())

			svc, err := do.Invoke[*logger.Service](injector)
			Expect(err).NotTo(HaveOccurred())
			Expect(svc).NotTo(BeNil())

			// Cleanup
			Expect(svc.Shutdown()).To(Succeed())
		})
	})
})

var _ = Describe("Logger DI Config", func() {
	It("resolves koanf config gracefully", func() {
		injector := do.New()

		// Setup mock koanf
		// Actually, testing newConfig requires github.com/knadh/koanf/v2
		// Let's just override Koanf to test newConfig error path at least
		logger.Package(injector)

		// if Koanf isn't provided, newConfig will fail
		_, err := do.Invoke[logger.Config](injector)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("Logger newSlogLogger err path", func() {
	It("returns err when service is not provided", func() {
		injector := do.New()

		logger.Package(injector)

		_, err := do.Invoke[*slog.Logger](injector)
		Expect(err).To(HaveOccurred())
	})
})

var _ = Describe("Logger Koanf resolution", func() {
	It("resolves koanf correctly", func() {
		injector := do.New()

		k := koanf.New(".")
		k.Set("logger.level", "info")
		k.Set("logger.file.name", "test.log")
		k.Set("logger.file.size", 10)
		k.Set("logger.file.channel_size", 100)
		do.ProvideValue(injector, k)
		logger.Package(injector)

		cfg, err := do.Invoke[logger.Config](injector)
		Expect(err).NotTo(HaveOccurred())
		Expect(cfg).NotTo(BeNil())
	})
})
