package domain

import (
	"errors"
	"testing"
)

func TestWrap(t *testing.T) {
	mainErr := errors.New("main error")
	subErr := errors.New("sub error")

	err := Wrap(mainErr, subErr)

	if !errors.Is(err, mainErr) {
		t.Errorf("expected error to wrap main error")
	}

	if !errors.Is(err, subErr) {
		t.Errorf("expected error to wrap sub error")
	}

	expectedStr := "main error: sub error"
	if err.Error() != expectedStr {
		t.Errorf("expected error string %q, got %q", expectedStr, err.Error())
	}
}

func TestErrInvalidArgument(t *testing.T) {
	subErr := errors.New("invalid id")

	err := ErrInvalidArgument(subErr)

	if !errors.Is(err, ErrValidation) {
		t.Errorf("expected error to wrap ErrValidation")
	}

	if !errors.Is(err, subErr) {
		t.Errorf("expected error to wrap sub error")
	}

	expectedStr := "domain: validation error: invalid id"
	if err.Error() != expectedStr {
		t.Errorf("expected error string %q, got %q", expectedStr, err.Error())
	}
}
