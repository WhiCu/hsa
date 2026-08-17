package idgen

import (
	"crypto/rand"
	"io"
	"sync"
	"unsafe"

	"github.com/google/uuid"
)

const batchSize = 256 // 256 * 16 = 4096

type batchBuffer struct {
	uuids [batchSize]uuid.UUID
	idx   int
}

func (b *batchBuffer) refill() error {
	raw := unsafe.Slice((*byte)(unsafe.Pointer(&b.uuids[0])), batchSize*16)
	if _, err := io.ReadFull(rand.Reader, raw); err != nil {
		return err
	}

	for i := range batchSize {
		b.uuids[i][6] = (b.uuids[i][6] & 0x0f) | 0x40 // RFC 4122 Version 4
		b.uuids[i][8] = (b.uuids[i][8] & 0x3f) | 0x80 // RFC 4122 Variant
	}

	b.idx = 0
	return nil
}

type PooledGenerator struct {
	pool sync.Pool
}

func NewPooledGenerator() *PooledGenerator {
	g := &PooledGenerator{
		pool: sync.Pool{
			New: func() any {
				return &batchBuffer{idx: batchSize}
			},
		},
	}
	return g
}

func (g *PooledGenerator) Pool() *sync.Pool {
	return &g.pool
}

func (g *PooledGenerator) NewID() uuid.UUID {
	buf, ok := g.pool.Get().(*batchBuffer)
	// uncovered: defensive sync.Pool assertion, we only put valid *batchBuffers
	if !ok {
		return uuid.New()
	}

	if buf.idx >= batchSize {
		if err := buf.refill(); err != nil {
			g.pool.Put(buf)
			return uuid.New()
		}
	}

	id := buf.uuids[buf.idx]
	buf.idx++

	g.pool.Put(buf)
	return id
}
