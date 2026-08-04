package application

import (
	"testing"

	"github.com/whicu/hsa/internal/domain/key"
)

func TestWrappedKeyInput_String(t *testing.T) {
	wki := WrappedKeyInput{
		Scope:         key.ScopeMain,
		WrappedDEK:    []byte("my-secret-dek"),
		WrapAlgorithm: "AES-256-GCM",
		ViaRecovery:   true,
	}

	str := wki.String()
	expected := "WrappedKeyInput{Scope: 0, WrappedDEK: ***REDACTED***, WrapAlgorithm: AES-256-GCM, ViaRecovery: true}"

	if str != expected {
		t.Errorf("Expected %s, got: %s", expected, str)
	}
}
