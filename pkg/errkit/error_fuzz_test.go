package errkit_test

import (
	"errors"
	"testing"
	"github.com/whicu/hsa/pkg/errkit"
)

func FuzzAppendMutates(f *testing.F) {
	f.Add("a", "b")
	f.Fuzz(func(t *testing.T, s1, s2 string) {
		err1 := errors.New(s1)
		err2 := errors.New(s2)

		base := errkit.Append(nil, err1)

		// base1 and base2 should be distinct
		_ = errkit.Append(base, err2)
		_ = errkit.Append(base, errors.New("c"))

		// wait, if it mutates, base now has BOTH err2 and "c"
		unwrapped := base.Unwrap()
		if len(unwrapped) > 1 {
			t.Errorf("base was mutated! len is %d", len(unwrapped))
		}
	})
}
