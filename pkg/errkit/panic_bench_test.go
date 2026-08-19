package errkit

import (
	"errors"
	"testing"
)

func BenchmarkPanicError_Error_String(b *testing.B) {
	e := NewPanicError("something went wrong")

	for b.Loop() {
		_ = e.Error()
	}
}

func BenchmarkPanicError_Error_Error(b *testing.B) {
	e := NewPanicError(errors.New("something went wrong"))

	for b.Loop() {
		_ = e.Error()
	}
}
