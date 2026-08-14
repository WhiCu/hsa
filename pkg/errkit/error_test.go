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

func TestAppend_Codecov(t *testing.T) {
	err1 := errors.New("err1")
	err2 := errors.New("err2")

	// Test nil *Error handling
	var nilErr *Error = nil
	appendedToNil := Append(nilErr, err1)
	if appendedToNil == nil {
		t.Errorf("expected appended error not to be nil")
	}

	// Test appending an *Error to an *Error
	baseErr := Append(err1)
	toAppend := Append(err2)
	merged := Append(baseErr, toAppend)
	if len(merged.Errors) != 2 {
		t.Errorf("expected 2 errors, got %d", len(merged.Errors))
	}

	// Test appending nil to an *Error
	mergedNil := Append(baseErr, nil)
	if len(mergedNil.Errors) != 1 {
		t.Errorf("expected 1 errors, got %d", len(mergedNil.Errors))
	}

	// Test unwrapping
	if len(merged.Unwrap()) != 2 {
		t.Errorf("expected unwrapped length to be 2")
	}
	if len(nilErr.Unwrap()) != 0 {
		t.Errorf("expected nil unwrapped length to be 0")
	}

	// Test error string
	if len(merged.Error()) == 0 {
		t.Errorf("expected error string to not be empty")
	}

	// Test appending with nil base target
	var nilErr2 *Error
	appendedToNil2 := Append(nilErr2, toAppend)
	if appendedToNil2 == nil {
		t.Errorf("expected appendedToNil2 not to be nil")
	}

	// Test appending an *Error nil
	appendedToBase := Append(baseErr, nilErr2)
	if len(appendedToBase.Errors) != 1 {
		t.Errorf("expected appendedToBase errors to be 1")
	}
}

func TestAppend_MoreCoverage(t *testing.T) {
	// target, ok := errors.AsType[*Error](err) is false
	err1 := errors.New("err1")
	err2 := errors.New("err2")
	err3 := errors.New("err3")

	// Test normal error appending to normal error
	appended := Append(err1, err2)
	if appended == nil || len(appended.Errors) != 2 {
		t.Errorf("expected 2 errors")
	}

	appended2 := Append(nil, err1, err2)
	if appended2 == nil || len(appended2.Errors) != 2 {
		t.Errorf("expected 2 errors")
	}

	// e != nil but not *Error
	var eTarget *Error
	appended3 := Append(eTarget, err1, err2, err3)
	if appended3 == nil || len(appended3.Errors) != 3 {
		t.Errorf("expected 3 errors")
	}
}
