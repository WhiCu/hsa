package idgen_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/whicu/hsa/pkg/idgen"
)

func BenchmarkPooledGenerator_NewID(b *testing.B) {
	gen := idgen.NewPooledGenerator()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = gen.NewID()
		}
	})
}

func BenchmarkStandardUUID_New(b *testing.B) {
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = uuid.New()
		}
	})
}
