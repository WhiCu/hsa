package crypto_test

import (
	"testing"
	"github.com/whicu/hsa/internal/infrastructure/crypto"
)

func FuzzSecretManagerGenerateToken(f *testing.F) {
	sm := crypto.NewSecretManager([]byte("secret"))
	f.Add(32)
	f.Add(0)
	f.Add(-1)
	f.Add(100000)

	f.Fuzz(func(t *testing.T, n int) {
		token, hash, err := sm.GenerateToken(n)
		if n < 0 || n > 1<<20 {
			if err == nil {
				t.Errorf("expected error for invalid n (%d), got nil", n)
			}
			return
		}
		if err != nil {
			t.Errorf("unexpected error for n=%d: %v", n, err)
		}
		if len(token) == 0 && n > 0 {
			t.Errorf("expected token length > 0, got 0")
		}
		_ = hash
	})
}
