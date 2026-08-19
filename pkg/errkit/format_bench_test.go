package errkit

import (
	"errors"
	"testing"
)

func BenchmarkListFormatFunc_Single(b *testing.B) {
	errs := []error{errors.New("something went wrong")}

	for b.Loop() {
		_ = ListFormatFunc(errs)
	}
}

func BenchmarkListFormatFunc_Multiple(b *testing.B) {
	errs := []error{
		errors.New("first error"),
		errors.New("second error"),
		errors.New("third error"),
	}

	for b.Loop() {
		_ = ListFormatFunc(errs)
	}
}
