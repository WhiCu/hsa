package errkit

import "errors"

type Mapper[E error, T any] func(E) T
type Handler[T any] func(err error) (T, bool)
type Option[T any] func(*Registry[T])

type Registry[T any] struct {
	handlers []Handler[T]
	defaul   func(error) T
}

func New[T any](opts ...Option[T]) *Registry[T] {
	r := &Registry[T]{}
	for _, o := range opts {
		o(r)
	}
	return r
}

func (r *Registry[T]) OnAs[E error](m Mapper[E, T]) *Registry[T] {
	r.handlers = append(r.handlers, func(err error) (T, bool) {
		if e, ok := errors.AsType[E](err); ok {
			return m(e), true
		}
		var zero T
		return zero, false
	})
	return r
}

func (r *Registry[T]) OnIs(m Mapper[error, T], target error) *Registry[T] {
	r.handlers = append(r.handlers, func(err error) (T, bool) {
		if errors.Is(err, target) {
			return m(err), true
		}
		var zero T
		return zero, false
	})
	return r
}

func (r *Registry[T]) OnMatch(match func(error) bool, m func(error) T) *Registry[T] {
	r.handlers = append(r.handlers, func(err error) (T, bool) {
		if match(err) {
			return m(err), true
		}
		var zero T
		return zero, false
	})
	return r
}

func (r *Registry[T]) Default(m Mapper[error, T]) *Registry[T] {
	r.defaul = m
	return r
}
func (r *Registry[T]) Resolve(err error) (T, bool) {
	for _, h := range r.handlers {
		if v, ok := h(err); ok {
			return v, true
		}
	}
	if r.defaul != nil {
		return r.defaul(err), true
	}
	var zero T
	return zero, false
}

func (r *Registry[T]) Handle(err error) (T, error) {
	v, ok := r.Resolve(err)
	if !ok {
		return v, err
	}
	return v, nil
}
