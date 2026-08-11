package idgen

import (
	"context"

	"github.com/google/uuid"
)

type pooledGenerator struct {
	pool chan uuid.UUID
}

func NewPooledGenerator(ctx context.Context, poolSize int) *pooledGenerator {
	g := &pooledGenerator{
		pool: make(chan uuid.UUID, poolSize),
	}

	go g.worker(ctx)

	return g
}

func (g *pooledGenerator) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case g.pool <- uuid.New():
		}
	}
}

func (g *pooledGenerator) NewID() uuid.UUID {
	select {
	case id := <-g.pool:
		return id
	default:
		return uuid.New()
	}
}
