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

func OnAs[E error, T any](m Mapper[E, T]) Option[T] {
	return func(r *Registry[T]) {
		r.handlers = append(r.handlers, func(err error) (T, bool) {
			if e, ok := errors.AsType[E](err); ok {
				return m(e), true
			}
			var zero T
			return zero, false
		})
	}
}
func OnIs[T any](m Mapper[error, T], target error) Option[T] {
	return func(r *Registry[T]) {
		r.handlers = append(r.handlers, func(err error) (T, bool) {
			if errors.Is(err, target) {
				return m(err), true
			}
			var zero T
			return zero, false
		})
	}
}
func OnMatch[T any](match func(error) bool, m func(error) T) Option[T] {
	return func(r *Registry[T]) {
		r.handlers = append(r.handlers, func(err error) (T, bool) {
			if match(err) {
				return m(err), true
			}
			var zero T
			return zero, false
		})
	}
}
func Default[T any](m Mapper[error, T]) Option[T] {
	return func(r *Registry[T]) { r.defaul = m }
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
