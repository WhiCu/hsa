package errkit

import (
	"errors"
	"testing"
)

func TestListFormatFunc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		errs     []error
		expected string
	}{
		{
			name:     "single error",
			errs:     []error{errors.New("first error")},
			expected: "1 error occurred:\n\t* first error\n\n",
		},
		{
			name:     "multiple errors",
			errs:     []error{errors.New("first error"), errors.New("second error")},
			expected: "2 errors occurred:\n\t* first error\n\t* second error\n\n",
		},
		{
			name:     "empty errors",
			errs:     []error{},
			expected: "0 errors occurred:\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := ListFormatFunc(tt.errs)
			if result != tt.expected {
				t.Errorf("expected:\n%q\ngot:\n%q", tt.expected, result)
			}
		})
	}
}
