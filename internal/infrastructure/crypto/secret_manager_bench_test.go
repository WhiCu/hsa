package crypto

import (
	"crypto/hmac"
	"crypto/sha256"
	"hash"
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
