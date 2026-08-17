package idgen_test

import (
	"crypto/rand"
	"errors"
	"testing/iotest"
	"sync"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"

	"github.com/whicu/hsa/pkg/idgen"
)

var _ = Describe("Generator", func() {
	var (
		injector  do.Injector
		generator *idgen.PooledGenerator
	)

	BeforeEach(func() {
		injector = do.New(idgen.Package)

		var err error
		generator, err = do.Invoke[*idgen.PooledGenerator](injector)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("RFC 4122 Compliance", func() {
		It("generates a non-nil valid UUIDv4 with correct version and variant bits", func() {
			id := generator.NewID()

			Expect(id).NotTo(Equal(uuid.Nil))
			Expect(id.Version()).To(Equal(uuid.Version(4)))
			Expect(id.Variant()).To(Equal(uuid.RFC4122))

			Expect(id[6] & 0xf0).To(Equal(byte(0x40))) // Version 4
			Expect(id[8] & 0xc0).To(Equal(byte(0x80))) // Variant 10
		})
	})

	Describe("Batch Buffer Transition & Uniqueness", func() {
		It("seamlessly refills the buffer across multiple batch sizes without collisions", func() {
			// batchSize = 256, генерируем 256 * 4 + 10 = 1034 UUIDs для проверки переполнения буфера
			const totalIDs = 1034
			seen := make(map[uuid.UUID]struct{}, totalIDs)

			for i := range totalIDs {
				id := generator.NewID()

				Expect(id).NotTo(Equal(uuid.Nil))
				Expect(id.Version()).To(Equal(uuid.Version(4)))
				Expect(id.Variant()).To(Equal(uuid.RFC4122))

				_, exists := seen[id]
				Expect(exists).To(BeFalse(), "Collision detected at index %d: %s", i, id)
				seen[id] = struct{}{}
			}

			Expect(seen).To(HaveLen(totalIDs))
		})
	})

	Describe("Concurrent Generation (Thread Safety)", func() {
		It("generates unique UUIDs concurrently across multiple goroutines", func() {
			const (
				goroutines = 50
				idsPerG    = 500
				totalIDs   = goroutines * idsPerG
			)

			var wg sync.WaitGroup
			idChan := make(chan uuid.UUID, totalIDs)

			for range goroutines {
				wg.Go(func() {
					for range idsPerG {
						idChan <- generator.NewID()
					}
				})
			}

			wg.Wait()
			close(idChan)

			seen := make(map[uuid.UUID]struct{}, totalIDs)
			for id := range idChan {
				Expect(id).NotTo(Equal(uuid.Nil))
				Expect(id.Version()).To(Equal(uuid.Version(4)))
				Expect(id.Variant()).To(Equal(uuid.RFC4122))

				_, exists := seen[id]
				Expect(exists).To(BeFalse(), "Concurrent collision detected: %s", id)
				seen[id] = struct{}{}
			}

			Expect(seen).To(HaveLen(totalIDs))
		})
	})
	Describe("Error Fallbacks", func() {		It("handles bad pool elements by falling back", func() {
			gen := idgen.NewPooledGenerator()
			gen.Pool().Put("not a buffer")
			id := gen.NewID()
			Expect(id).NotTo(Equal(uuid.Nil))
		})

		It("handles random source errors by falling back", func() {
			oldReader := rand.Reader
			defer func() { rand.Reader = oldReader }()
			rand.Reader = iotest.ErrReader(errors.New("simulated random error"))
			gen := idgen.NewPooledGenerator()
			id := gen.NewID()
			Expect(id).NotTo(Equal(uuid.Nil))
			Expect(id.Version()).To(Equal(uuid.Version(4)))
		})

	})
})
