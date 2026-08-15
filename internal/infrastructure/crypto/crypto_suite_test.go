package crypto_test

import (
	"testing"

	"aidanwoods.dev/go-paseto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/whicu/hsa/internal/infrastructure/crypto"
)

func TestCrypto(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Crypto Suite")
}

func newTestCryptoConfig() crypto.Config {
	symKey := paseto.NewV4SymmetricKey()
	asymKey := paseto.NewV4AsymmetricSecretKey()

	cfg := crypto.Config{
		HMACSecret: "a-very-secure-random-hmac-secret-string-at-least-32-bytes",
		PASETO: crypto.PASETOConfig{
			SymmetricKey: symKey.ExportHex(),
			SecretKey:    asymKey.ExportHex(),
			PublicKey:    asymKey.Public().ExportHex(),
		},
	}
	return cfg
}

var testConfig = newTestCryptoConfig()
