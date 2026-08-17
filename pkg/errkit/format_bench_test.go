package errkit

import (
	"errors"
	"testing"
)

func BenchmarkListFormatFunc_Single(b *testing.B) {
	errs := []error{errors.New("something went wrong")}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ListFormatFunc(errs)
	}
}

func BenchmarkListFormatFunc_Multiple(b *testing.B) {
	errs := []error{
		errors.New("first error"),
		errors.New("second error"),
		errors.New("third error"),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ListFormatFunc(errs)
	}
}
