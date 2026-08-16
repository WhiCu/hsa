package idgen

import (
	"github.com/samber/do/v2"
)

func newPooledGenerator(_ do.Injector) (*PooledGenerator, error) {
	return NewPooledGenerator(), nil
}

var Package = do.Package(
	do.Lazy(newPooledGenerator),
)
