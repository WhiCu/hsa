package webauthnadapter

import (
	"context"
	"testing"
	"time"
	"log/slog"

	"github.com/google/uuid"
	"github.com/go-webauthn/webauthn/webauthn"
)

type mockChallenge struct{}

func (m *mockChallenge) Encode(payload any, ttl time.Duration) (string, error) {
	return "token", nil
}

func (m *mockChallenge) Decode(token string, out any) error {
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
