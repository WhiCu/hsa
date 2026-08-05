package crypto_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/whicu/hsa/internal/infrastructure/crypto"
)

var _ = Describe("ChallengeCodec", func() {
	Context("SecretManager Helper Functions", func() {
		It("String", func() {
			sm := crypto.NewSecretManager([]byte("my-secret-key"))
			str := sm.String()

			Expect(str).To(Equal("SecretManager{secretKey: ***REDACTED***}"))

			var nilSM *crypto.SecretManager
			Expect(nilSM.String()).To(Equal("<nil>"))
		})
	})
})
