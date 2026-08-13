package errkit

import (
	"errors"
	"testing"
)

func TestFuzz_AppendMutability(t *testing.T) {
	err1 := errors.New("err1")
	err2 := errors.New("err2")
	err3 := errors.New("err3")

	base := Append(err1)

	// Two separate paths appending to the same base error
	newErr1 := Append(base, err2)
	newErr2 := Append(base, err3)

	_ = newErr1
	_ = newErr2

	if len(base.Errors) != 1 {
		t.Errorf("base error was mutated! Expected len 1, got %d", len(base.Errors))
	}

	if len(newErr1.Errors) != 2 {
		t.Errorf("newErr1 len incorrect! Expected 2, got %d", len(newErr1.Errors))
	}

	if len(newErr2.Errors) != 2 {
		t.Errorf("newErr2 len incorrect! Expected 2, got %d", len(newErr2.Errors))
	}
}
