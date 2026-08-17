package crypto_test

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/whicu/hsa/internal/infrastructure/crypto"
)

var _ = Describe("Config", func() {
	Describe("String", func() {
		It("redacts sensitive fields in Config", func() {
			cfg := crypto.Config{
				HMACSecret: "super-secret-hmac-key",
				PASETO: crypto.PASETOConfig{
					SymmetricKey: "symmetric-key-hex",
					SecretKey:    "secret-key-hex",
					PublicKey:    "public-key-hex",
				},
			}

			str := fmt.Sprintf("%+v", cfg)

			Expect(str).To(Equal("crypto.Config{HMACSecret: ***REDACTED***, PASETO: crypto.PASETOConfig{SymmetricKey: ***REDACTED***, SecretKey: ***REDACTED***, PublicKey: ***REDACTED***}}"))
			Expect(str).NotTo(ContainSubstring("super-secret-hmac-key"))
			Expect(str).NotTo(ContainSubstring("symmetric-key-hex"))
			Expect(str).NotTo(ContainSubstring("secret-key-hex"))
			Expect(str).NotTo(ContainSubstring("public-key-hex"))
		})

		It("redacts sensitive fields in PASETOConfig directly", func() {
			cfg := crypto.PASETOConfig{
				SymmetricKey: "symmetric-key-hex",
				SecretKey:    "secret-key-hex",
				PublicKey:    "public-key-hex",
			}

			str := fmt.Sprintf("%+v", cfg)

			Expect(str).To(Equal("crypto.PASETOConfig{SymmetricKey: ***REDACTED***, SecretKey: ***REDACTED***, PublicKey: ***REDACTED***}"))
			Expect(str).NotTo(ContainSubstring("symmetric-key-hex"))
			Expect(str).NotTo(ContainSubstring("secret-key-hex"))
			Expect(str).NotTo(ContainSubstring("public-key-hex"))
		})
	})
})
