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
	// ⚡ Bolt: Use type assertion and string concatenation for common panic types (error and string)
	// instead of fmt.Sprintf to avoid reflection overhead and reduce memory allocations.
	if err, ok := e.Value.(error); ok {
		return "panic recovered: " + err.Error()
	}
	if str, ok := e.Value.(string); ok {
		return "panic recovered: " + str
	}
	return fmt.Sprintf("panic recovered: %v", e.Value)
}

func (e *PanicError) Unwrap() error {
	if err, ok := e.Value.(error); ok {
		return err
	}
	return nil
}
