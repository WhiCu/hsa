package crypto_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"

	"github.com/whicu/hsa/internal/infrastructure/crypto"
)

var _ = Describe("SecretManager", func() {
	var (
		injector do.Injector
		sm       *crypto.SecretManager
	)

	BeforeEach(func() {

		injector = do.New(crypto.Package)
		do.OverrideValue(injector, testConfig)

		var err error
		sm, err = do.Invoke[*crypto.SecretManager](injector)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("GenerateHash", func() {
		It("generates deterministic HMAC hash in Base32", func() {
			raw := "my-secret-token-value"

			hash1, err := sm.GenerateHash(raw)
			Expect(err).NotTo(HaveOccurred())
			Expect(hash1).NotTo(BeEmpty())

			hash2, err := sm.GenerateHash(raw)
			Expect(err).NotTo(HaveOccurred())
			Expect(hash1).To(Equal(hash2))
		})

		It("produces different hashes for different inputs", func() {
			hash1, err := sm.GenerateHash("raw-token-1")
			Expect(err).NotTo(HaveOccurred())

			hash2, err := sm.GenerateHash("raw-token-2")
			Expect(err).NotTo(HaveOccurred())

			Expect(hash1).NotTo(Equal(hash2))
		})
	})

	Describe("GenerateToken", func() {
		It("generates random Base32 token and corresponding hash", func() {
			token, hash, err := sm.GenerateToken(32)
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())
			Expect(hash).NotTo(BeEmpty())

			expectedHash, err := sm.GenerateHash(token)
			Expect(err).NotTo(HaveOccurred())
			Expect(hash).To(Equal(expectedHash))
		})

		It("returns error for invalid token length", func() {
			_, _, err := sm.GenerateToken(-1)
			Expect(err).To(HaveOccurred())

			_, _, err = sm.GenerateToken((1 << 20) + 1)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("String Redaction", func() {
		It("redacts secretKey in logs", func() {
			Expect(sm.String()).To(Equal("SecretManager{secretKey: ***REDACTED***}"))

			var nilSM *crypto.SecretManager
			Expect(nilSM.String()).To(Equal("<nil>"))
		})
	})
})

func FuzzSecretManagerGenerateToken(f *testing.F) {
	sm := crypto.NewSecretManager([]byte("secret-key-for-fuzzing-32-bytes!"))
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
