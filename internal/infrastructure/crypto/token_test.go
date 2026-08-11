package crypto_test

import (
	"aidanwoods.dev/go-paseto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/whicu/hsa/internal/infrastructure/crypto"
)

var _ = Describe("Token Security Stringers", func() {
	It("redacts keys in TokenCodec", func() {
		codec := crypto.NewTokenCodec(paseto.NewV4SymmetricKey())
		Expect(codec.String()).To(Equal("TokenCodec{privateKey: ***REDACTED***}"))

		var nilCodec *crypto.TokenCodec
		Expect(nilCodec.String()).To(Equal("<nil>"))
	})

	It("redacts keys in AccessTokenIssuer", func() {
		secretKey := paseto.NewV4AsymmetricSecretKey()
		issuer := crypto.NewAccessTokenIssuer(secretKey)
		Expect(issuer.String()).To(Equal("AccessTokenIssuer{secretKey: ***REDACTED***}"))

		var nilIssuer *crypto.AccessTokenIssuer
		Expect(nilIssuer.String()).To(Equal("<nil>"))
	})

	It("redacts keys in AccessTokenVerifier", func() {
		secretKey := paseto.NewV4AsymmetricSecretKey()
		verifier := crypto.NewAccessTokenVerifier(secretKey.Public())
		Expect(verifier.String()).To(Equal("AccessTokenVerifier{publicKey: ***REDACTED***}"))

		var nilVerifier *crypto.AccessTokenVerifier
		Expect(nilVerifier.String()).To(Equal("<nil>"))
	})
})
