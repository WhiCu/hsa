package crypto_test

import (
	"github.com/whicu/hsa/internal/infrastructure/crypto"
	"aidanwoods.dev/go-paseto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("AccessTokenVerifier", func() {
	It("String method redacts publicKey", func() {
		sk := paseto.NewV4AsymmetricSecretKey()
		pk := sk.Public()
		tv := crypto.NewAccessTokenVerifier(pk)
		Expect(tv.String()).To(Equal("AccessTokenVerifier{publicKey: ***REDACTED***}"))

		var nilTv *crypto.AccessTokenVerifier
		Expect(nilTv.String()).To(Equal("<nil>"))
	})
})
