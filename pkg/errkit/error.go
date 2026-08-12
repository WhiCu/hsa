package errkit

import "errors"

type Error struct {
	Errors      []error
	ErrorFormat ErrorFormatFunc
}

func (e *Error) Error() string {
	fn := e.ErrorFormat
	if fn == nil {
		fn = ListFormatFunc
	}

	return fn(e.Errors)
}

func Append(err error, errs ...error) *Error {
	if target, ok := errors.AsType[*Error](err); ok {
		if target == nil {
			target = &Error{}
		}
		for _, e := range errs {
			if eTarget, okType := errors.AsType[*Error](e); okType {
				if eTarget != nil {
					target.Errors = append(target.Errors, eTarget.Errors...)
				}
			} else if e != nil {
				target.Errors = append(target.Errors, e)
			}
		}

		return target
	}

	// PERFORMANCE: Avoid recursive call and extra slice allocation by
	// allocating the exact capacity and unrolling the append loop
	target := &Error{
		Errors: make([]error, 0, len(errs)+1),
	}
	if err != nil {
		target.Errors = append(target.Errors, err)
	}
	for _, e := range errs {
		if eTarget, okType := errors.AsType[*Error](e); okType {
			if eTarget != nil {
				target.Errors = append(target.Errors, eTarget.Errors...)
			}
		} else if e != nil {
			target.Errors = append(target.Errors, e)
		}
	}

	return target
}

func (e *Error) Unwrap() []error {
	if e == nil || len(e.Errors) == 0 {
		return nil
	}

	return e.Errors
}
