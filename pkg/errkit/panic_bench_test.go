package errkit

import (
	"errors"
	"testing"
)

func BenchmarkPanicError_Error_String(b *testing.B) {
	e := NewPanicError("something went wrong")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.Error()
	}
}

func BenchmarkPanicError_Error_Error(b *testing.B) {
	e := NewPanicError(errors.New("something went wrong"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = e.Error()
	}
}
