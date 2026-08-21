package storage_test

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"
	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/whicu/hsa/internal/infrastructure/storage"
	"github.com/whicu/hsa/pkg/logger"
	"github.com/whicu/hsa/test/testutil"
)

var (
	globalConfig storage.Config
)

func BuildStorageConfig(ctx context.Context, ctr *postgres.PostgresContainer) (storage.Config, error) {
	host, err := ctr.Host(ctx)
	if err != nil {
		return storage.Config{}, fmt.Errorf("get container host: %w", err)
	}
	port, err := ctr.MappedPort(ctx, "5432/tcp")
	if err != nil {
		return storage.Config{}, fmt.Errorf("get mapped port: %w", err)
	}

	return storage.Config{
		Host:            host,
		Port:            port.Port(),
		User:            testutil.TestDBUser,
		Pass:            testutil.TestDBPass,
		Name:            testutil.TestDBName,
		Insecure:        true,
		MaxOpenConns:    10,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
	}, nil
}

var _ = BeforeSuite(func(ctx SpecContext) {
	ctr, err := testutil.StartPostgresContainer(ctx)
	Expect(err).ToNot(HaveOccurred())

	globalConfig, err = BuildStorageConfig(ctx, ctr)
	Expect(err).ToNot(HaveOccurred())

	DeferCleanup(func(ctx SpecContext) {
		Expect(ctr.Terminate(ctx)).To(Succeed())
	})
})

var _ = Describe("Storage", func() {
	var (
		injector do.Injector
		srg      *storage.Storage
	)

	BeforeEach(func(ctx SpecContext) {
		injector = do.New(storage.Package)

		do.OverrideValue(injector, logger.NewNOPSlog())
		do.OverrideValue(injector, globalConfig)
		do.OverrideValue[context.Context](injector, ctx)

		var err error
		srg, err = do.Invoke[*storage.Storage](injector)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func(_ SpecContext) {
			rep := injector.Shutdown()
			Expect(rep.Succeed).To(BeTrue())
		})
	})

	Describe("Ping", func() {
		It("successfully pings active database connection", func(ctx SpecContext) {
			Expect(srg.Ping(ctx)).To(Succeed())
		})
	})

	Describe("Up", func() {
		It("successfully executes all pending migrations", func(ctx SpecContext) {
			Expect(srg.Up(ctx)).To(Succeed())
		})
	})

	Describe("Reset", func() {
		It("successfully resets and reapplies database schema", func(ctx SpecContext) {
			Expect(srg.Up(ctx)).To(Succeed())
			Expect(srg.Reset(ctx)).To(Succeed())
		})
	})

	Describe("Shutdown", func() {
		Context("Order 1: Direct Storage.Shutdown() then injector.Shutdown()", func() {
			It("closes database pool explicitly before injector shutdown", func(ctx SpecContext) {
				srg.Shutdown()
				Expect(srg.Ping(ctx)).To(HaveOccurred())
				Expect(injector.ShutdownWithContext(ctx).Succeed).To(BeTrue())
			})
		})

		Context("Order 2: injector.Shutdown() then Direct Storage.Shutdown()", func() {
			It("shuts down DI container before explicit Storage.Shutdown()", func(ctx SpecContext) {
				Expect(injector.ShutdownWithContext(ctx).Succeed).To(BeTrue())
				Expect(srg.Ping(ctx)).To(HaveOccurred())
				Expect(func() { srg.Shutdown() }).ToNot(Panic())
				Expect(srg.Ping(ctx)).To(HaveOccurred())
			})
		})
	})
})
