# Fix revive issues and goimports in webauthn adapter
cat << 'INNEREOF' > internal/infrastructure/auth/webauthn/registrator_fuzz_test.go
package webauthnadapter

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

type mockChallenge struct{}

func (m *mockChallenge) Encode(_ any, _ time.Duration) (string, error) {
	return "token", nil
}

func (m *mockChallenge) Decode(_ string, _ any) error {
	return nil
}

func TestFuzz_Registrator_NilUser(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Registrator.Begin panicked: %v", r)
		}
	}()

	wa, _ := webauthn.New(&webauthn.Config{
		RPDisplayName: "Test RP",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost"},
	})

	reg := NewRegistrator(slog.Default(), wa, &mockChallenge{}, time.Minute)

	ctx := context.Background()
	_, _, _ = reg.Begin(ctx, uuid.Nil, uuid.Nil)
}
INNEREOF

# Fix gosec G602 in errkit/format.go by bounds checking (just in case)
cat << 'INNEREOF' > pkg/errkit/format.go
package errkit

import (
	"fmt"
	"strings"
)

type ErrorFormatFunc func([]error) string

func ListFormatFunc(es []error) string {
	if len(es) == 0 {
		return ""
	}
	if len(es) == 1 {
		return fmt.Sprintf("1 error occurred:\n\t* %s\n\n", es[0])
	}

	points := make([]string, len(es))
	for i, err := range es {
		points[i] = fmt.Sprintf("* %s", err)
	}

	return fmt.Sprintf(
		"%d errors occurred:\n\t%s\n\n",
		len(es), strings.Join(points, "\n\t"))
}
INNEREOF

# Remove the ViaRecovery field in the test, since it doesn't exist on WrappedKeyInput struct in main codebase
cat << 'INNEREOF' > internal/application/finish_invite_registration_added_test.go
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
	}

	str := wki.String()
	expected := "WrappedKeyInput{Scope: 0, WrappedDEK: ***REDACTED***, WrapAlgorithm: AES-256-GCM, ViaRecovery: false}"

	if str != expected {
		t.Errorf("Expected %s, got: %s", expected, str)
	}
}
INNEREOF
