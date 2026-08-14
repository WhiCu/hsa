package errkit_test

import (
	"errors"
	"fmt"
	"testing"
	"github.com/whicu/hsa/pkg/errkit"
)

func TestFuzz_AppendMutation(t *testing.T) {
	err1 := errors.New("err1")
	err2 := errors.New("err2")

	baseErr := errkit.Append(nil, err1) // Error{Errors: [err1]}
	wrappedBase := fmt.Errorf("wrapped: %w", baseErr)

	_ = errkit.Append(wrappedBase, err2)

	// Does wrappedBase's underlying error get mutated?
	if len(baseErr.Errors) > 1 {
		t.Errorf("expected baseErr to have 1 error, got %d", len(baseErr.Errors))
	}
}

func TestFuzz_AppendNil(t *testing.T) {
    var err error
    appended := errkit.Append(err, nil, nil)
    if appended == nil {
        t.Fatal("expected non-nil error")
    }
    if len(appended.Errors) != 0 {
        t.Fatalf("expected 0 errors, got %d", len(appended.Errors))
    }
}
