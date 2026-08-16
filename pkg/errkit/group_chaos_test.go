package errkit

import (
	"errors"
	"testing"
)

func TestFuzz_GroupPanicHandling(t *testing.T) {
	g := &Group{}
	g.Go(func() error {
		panic("chaos panic")
	})

	err := g.Wait()
	if err == nil {
		t.Fatal("expected panic to be recovered and returned as error")
	}

	var panicErr *PanicError
	if !errors.As(err, &panicErr) {
		t.Fatalf("expected error to be of type *PanicError, got %T", err)
	}

	if panicErr.Value != "chaos panic" {
		t.Fatalf("expected panic value to be 'chaos panic', got %v", panicErr.Value)
	}
}

func TestFuzz_GroupFinallyPanicHandling(t *testing.T) {
	g := &Group{}
	g.Finally(func() error {
		panic("chaos panic in finally")
	})

	err := g.Wait()
	if err == nil {
		t.Fatal("expected panic to be recovered and returned as error")
	}

	var panicErr *PanicError
	if !errors.As(err, &panicErr) {
		t.Fatalf("expected error to be of type *PanicError, got %T", err)
	}

	if panicErr.Value != "chaos panic in finally" {
		t.Fatalf("expected panic value to be 'chaos panic in finally', got %v", panicErr.Value)
	}
}
