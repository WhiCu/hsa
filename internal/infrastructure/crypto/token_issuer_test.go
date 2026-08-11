package crypto_test

import (
	"github.com/whicu/hsa/internal/infrastructure/crypto"
	"aidanwoods.dev/go-paseto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AccessTokenIssuer", func() {
	It("String method redacts secretKey", func() {
		sk := paseto.NewV4AsymmetricSecretKey()
		ti := crypto.NewAccessTokenIssuer(sk)
		Expect(ti.String()).To(Equal("AccessTokenIssuer{secretKey: ***REDACTED***}"))

		var nilTi *crypto.AccessTokenIssuer
		Expect(nilTi.String()).To(Equal("<nil>"))
	})
})
