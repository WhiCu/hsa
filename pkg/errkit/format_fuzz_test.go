package errkit

import (
	"testing"
)

func TestFuzz_ListFormatFunc_NilError(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("ListFormatFunc panicked with nil error: %v", r)
		}
	}()

	// Try empty
	ListFormatFunc(nil)
	ListFormatFunc([]error{})

	// Try slice with nil error
	ListFormatFunc([]error{nil})
	ListFormatFunc([]error{nil, nil})
}
