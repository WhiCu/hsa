package application

import (
	"strings"
	"testing"

	"github.com/whicu/hsa/internal/domain/key"
)

func TestWrappedKeyInput_String(t *testing.T) {
	wki := WrappedKeyInput{
		Scope:         key.ScopeMain,
		WrappedDEK:    []byte("my-secret-dek"),
		WrapAlgorithm: "AES-256-GCM",
	}

	str := wki.String()

	if !strings.Contains(str, "***REDACTED***") {
		t.Errorf("Expected string to redact WrappedDEK, got: %s", str)
	}

	if !strings.Contains(str, "AES-256-GCM") {
		t.Errorf("Expected string to contain wrap algorithm, got: %s", str)
	}
}
