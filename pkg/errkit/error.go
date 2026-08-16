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
		var newTarget *Error
		if target == nil {
			newTarget = &Error{}
		} else {
			newTarget = &Error{
				ErrorFormat: target.ErrorFormat,
			}
			if target.Errors != nil {
				newTarget.Errors = make([]error, len(target.Errors))
				copy(newTarget.Errors, target.Errors)
			}
		}
		for _, e := range errs {
			if eTarget, okType := errors.AsType[*Error](e); okType {
				if eTarget != nil {
					newTarget.Errors = append(newTarget.Errors, eTarget.Errors...)
				}
			} else if e != nil {
				newTarget.Errors = append(newTarget.Errors, e)
			}
		}

		return newTarget
	}

	newErrs := make([]error, 0, len(errs)+1)
	if err != nil {
		newErrs = append(newErrs, err)
	}
	newErrs = append(newErrs, errs...)

	return Append(&Error{}, newErrs...)
}

func (e *Error) Unwrap() []error {
	if e == nil || len(e.Errors) == 0 {
		return nil
	}

	return e.Errors
}
