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
	targetBase := &Error{}

	if target, ok := errors.AsType[*Error](err); ok {
		if target != nil {
			// Shallow copy the base target (preserves ErrorFormat and any other fields)
			newTarget := *target
			targetBase = &newTarget

			// Deep copy the slice to avoid mutation side-effects
			targetBase.Errors = make([]error, len(target.Errors))
			copy(targetBase.Errors, target.Errors)
		}
	} else if err != nil {
		targetBase.Errors = append(targetBase.Errors, err)
	}

	for _, e := range errs {
		if eTarget, okType := errors.AsType[*Error](e); okType {
			if eTarget != nil {
				targetBase.Errors = append(targetBase.Errors, eTarget.Errors...)
			}
		} else if e != nil {
			targetBase.Errors = append(targetBase.Errors, e)
		}
	}

	return targetBase
}

func (e *Error) Unwrap() []error {
	if e == nil || len(e.Errors) == 0 {
		return nil
	}

	return e.Errors
}
