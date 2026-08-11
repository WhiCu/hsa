package errkit

import (
	"fmt"
	"runtime/debug"
)

type PanicError struct {
	Value any
	Stack []byte
}

func NewPanicError(v any) *PanicError {
	return &PanicError{
		Value: v,
		Stack: debug.Stack(),
	}
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("panic recovered: %v", e.Value)
}

func (e *PanicError) Unwrap() error {
	if err, ok := e.Value.(error); ok {
		return err
	}
	return nil
}
