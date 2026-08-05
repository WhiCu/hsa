package errkit

import (
	"testing"
)

func TestFuzz_Group_NilFunctionGo(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Group.Go panicked with nil function: %v", r)
		}
	}()

	var g Group
	g.Go(nil)
	_ = g.Wait()
}

func TestFuzz_Group_NilFunctionFinally(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Group.Wait panicked with nil finally function: %v", r)
		}
	}()

	var g Group
	g.Finally(nil)
	_ = g.Wait()
}
