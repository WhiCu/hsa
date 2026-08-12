package errkit_test

import (
	"testing"

	"github.com/whicu/hsa/pkg/errkit"
)

func TestAppend_TypedNil(t *testing.T) {
	var typedNil *errkit.Error

	// Create an error interface that holds a typed nil
	var err error = typedNil

	appended := errkit.Append(err, nil)

	if len(appended.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(appended.Errors))
	}
}

func TestAppend_TypedNilInVariadic(t *testing.T) {
	var typedNil *errkit.Error

	var err error = typedNil

	// Base is a valid error, we append a typed nil
	base := errkit.Append(nil, errkit.Append(nil, nil))

	appended := errkit.Append(base, err)

	if len(appended.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(appended.Errors))
	}
}
