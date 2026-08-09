package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"hash"
	"strings"
	"sync"
	"testing"
)

var (
	secretKey  = []byte("super-secret-key-12345")
	dataToSign = []byte("Hello, World! This is test payload for HMAC SHA256 benchmarking.")
)

func BenchmarkHMAC_WithoutPool(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		h := hmac.New(sha256.New, secretKey)
		h.Write(dataToSign)
		_ = h.Sum(nil)
	}
}

func BenchmarkHMAC_WithPool(b *testing.B) {
	pool := &sync.Pool{
		New: func() any {
			return hmac.New(sha256.New, secretKey)
		},
	}

	b.ReportAllocs()

	for b.Loop() {
		h := pool.Get().(hash.Hash)
		h.Reset()
		h.Write(dataToSign)
		_ = h.Sum(nil)
		pool.Put(h)
	}
}

func BenchmarkHMAC_WithoutPool_Parallel(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			h := hmac.New(sha256.New, secretKey)
			h.Write(dataToSign)
			_ = h.Sum(nil)
		}
	})
}

func BenchmarkHMAC_WithPool_Parallel(b *testing.B) {
	pool := &sync.Pool{
		New: func() any {
			return hmac.New(sha256.New, secretKey)
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			h := pool.Get().(hash.Hash)
			h.Reset()
			h.Write(dataToSign)
			_ = h.Sum(nil)
			pool.Put(h)
		}
	})
}

func BenchmarkHMAC_ZeroAlloc(b *testing.B) {
	pool := &sync.Pool{
		New: func() any {
			return &hmacHolder{
				h: hmac.New(sha256.New, secretKey),
			}
		},
	}

	b.ReportAllocs()

	for b.Loop() {
		item := pool.Get().(*hmacHolder)

		item.h.Reset()
		item.h.Write(dataToSign)

		res := item.h.Sum(item.buf[:0])

		_ = res

		pool.Put(item)
	}
}

func BenchmarkHMAC_ZeroAlloc_Parallel(b *testing.B) {
	pool := &sync.Pool{
		New: func() any {
			return &hmacHolder{
				h: hmac.New(sha256.New, secretKey),
			}
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			item := pool.Get().(*hmacHolder)

			item.h.Reset()
			item.h.Write(dataToSign)
			res := item.h.Sum(item.buf[:0])

			_ = res

			pool.Put(item)
		}
	})
}

type testSecretManager struct {
	secretKey []byte
	hmacPool  sync.Pool
}

func newTestSecretManager() *testSecretManager {
	return &testSecretManager{
		secretKey: secretKey,
		hmacPool: sync.Pool{
			New: func() any {
				return &hmacHolder{
					h: hmac.New(sha256.New, secretKey),
				}
			},
		},
	}
}

func (sm *testSecretManager) generateHashBase32(raw string) (string, error) {
	v := sm.hmacPool.Get()
	item, ok := v.(*hmacHolder)
	if !ok {
		return "", errors.New("crypto: type assertion failed on hash pool")
	}
	defer sm.hmacPool.Put(item)
	item.h.Reset()

	_, err := item.h.Write([]byte(raw))
	if err != nil {
		return "", err
	}
	sum := item.h.Sum(item.buf[:0])
	encoded := b32.EncodeToString(sum)
	return encoded, nil
}

func (sm *testSecretManager) generateHashNaive(raw string) (string, error) {
	h := hmac.New(sha256.New, sm.secretKey)

	_, err := h.Write([]byte(raw))
	if err != nil {
		return "", err
	}
	sum := h.Sum(nil)
	encoded := b32.EncodeToString(sum)
	return encoded, nil
}

var benchInputs = map[string]string{
	"tiny_8B":      "abcdefgh",
	"small_32B":    strings.Repeat("a", 32),
	"medium_256B":  strings.Repeat("a", 256),
	"large_4KB":    strings.Repeat("a", 4096),
	"huge_32KB":    strings.Repeat("a", 32768),
	"gigantic_1MB": strings.Repeat("a", 1<<20),
}

func BenchmarkGenerateHash_Pooled(b *testing.B) {
	sm := newTestSecretManager()
	for name, input := range benchInputs {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := sm.generateHashBase32(input); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkGenerateHash_Naive(b *testing.B) {
	sm := newTestSecretManager()
	for name, input := range benchInputs {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := sm.generateHashNaive(input); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkGenerateHash_Pooled_Parallel(b *testing.B) {
	sm := newTestSecretManager()
	for name, input := range benchInputs {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if _, err := sm.generateHashBase32(input); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

func BenchmarkGenerateHash_Naive_Parallel(b *testing.B) {
	sm := newTestSecretManager()
	for name, input := range benchInputs {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					if _, err := sm.generateHashNaive(input); err != nil {
						b.Fatal(err)
					}
				}
			})
		})
	}
}

func BenchmarkGenerateToken_Pooled(b *testing.B) {
	sm := newTestSecretManager()
	b.ReportAllocs()
	for b.Loop() {
		raw, err := generateRandomBase32(32)
		if err != nil {
			b.Fatal(err)
		}
		if _, err = sm.generateHashBase32(raw); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateToken_Naive(b *testing.B) {
	sm := newTestSecretManager()
	b.ReportAllocs()
	for b.Loop() {
		raw, err := generateRandomBase32(32)
		if err != nil {
			b.Fatal(err)
		}
		if _, err = sm.generateHashNaive(raw); err != nil {
			b.Fatal(err)
		}
	}
}

func TestGenerateHash_PooledMatchesNaive(t *testing.T) {
	sm := newTestSecretManager()
	for name, input := range benchInputs {
		pooled, err := sm.generateHashBase32(input)
		if err != nil {
			t.Fatalf("%s: pooled error: %v", name, err)
		}
		naive, err := sm.generateHashNaive(input)
		if err != nil {
			t.Fatalf("%s: naive error: %v", name, err)
		}
		if pooled != naive {
			t.Fatalf("%s: mismatch: pooled=%s naive=%s", name, pooled, naive)
		}
	}
}
