package errkit

import (
	"errors"
)

type Mapper[E error, T any] func(error) T

type Registry[T any] struct {
	mappers       []func(err error) (T, bool)
	defaultMapper func(error) T
}

func Register[E error, T any](r *Registry[T], m Mapper[E, T]) {
	r.mappers = append(r.mappers, func(err error) (T, bool) {
		if e, ok := errors.AsType[E](err); ok {
			return m(e), true
		}

		var zero T
		return zero, false
	})
}

func RegisterDefault[T any](r *Registry[T], m Mapper[error, T]) {
	r.defaultMapper = m
}

func (r *Registry[T]) Resolve(err error) T {
	for _, m := range r.mappers {
		if v, ok := m(err); ok {
			return v
		}
	}

	if r.defaultMapper != nil {
		return r.defaultMapper(err)
	}

	var zero T
	return zero
}
