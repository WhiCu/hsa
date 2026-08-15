package crypto_test

import (
	"time"

	"aidanwoods.dev/go-paseto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"

	"github.com/whicu/hsa/internal/infrastructure/crypto"
)

type testPayload struct {
	ID   string `json:"id"`
	Data string `json:"data"`
}

var _ = Describe("TokenCodec", func() {
	var (
		injector     do.Injector
		codec        *crypto.TokenCodec
		pasetoKey    paseto.V4SymmetricKey
		validPayload testPayload
	)

	BeforeEach(func() {

		injector = do.New(crypto.Package)
		do.OverrideValue(injector, testConfig)

		var err error
		codec, err = do.Invoke[*crypto.TokenCodec](injector)
		Expect(err).NotTo(HaveOccurred())

		pasetoKey, err = paseto.V4SymmetricKeyFromHex(testConfig.PASETO.SymmetricKey)
		Expect(err).NotTo(HaveOccurred())

		validPayload = testPayload{
			ID:   "challenge-123",
			Data: "user-session-payload",
		}
	})

	Describe("Encode", func() {
		It("successfully encodes a valid payload into PASETO v4 local token", func() {
			tokenStr, err := codec.Encode(validPayload, time.Minute)

			Expect(err).NotTo(HaveOccurred())
			Expect(tokenStr).NotTo(BeEmpty())
			Expect(tokenStr).To(HavePrefix("v4.local."))
		})

		It("fails encoding when payload cannot be marshaled to JSON", func() {
			unmarshalablePayload := make(chan int)

			tokenStr, err := codec.Encode(unmarshalablePayload, time.Minute)

			Expect(err).To(HaveOccurred())
			Expect(tokenStr).To(BeEmpty())
		})
	})

	Describe("Decode", func() {
		It("successfully decodes a valid token into struct", func() {
			tokenStr, err := codec.Encode(validPayload, time.Minute)
			Expect(err).NotTo(HaveOccurred())

			var decoded testPayload
			err = codec.Decode(tokenStr, &decoded)

			Expect(err).NotTo(HaveOccurred())
			Expect(decoded).To(Equal(validPayload))
		})

		It("returns ErrTokenExpired when decoding an expired token", func() {
			tokenStr, err := codec.Encode(validPayload, -time.Minute)
			Expect(err).NotTo(HaveOccurred())

			var decoded testPayload
			err = codec.Decode(tokenStr, &decoded)

			Expect(err).To(MatchError(crypto.ErrTokenExpired))
		})

		It("returns ErrTokenMalformed for an invalid token string", func() {
			var decoded testPayload
			err := codec.Decode("invalid-paseto-token-string", &decoded)

			Expect(err).To(MatchError(crypto.ErrTokenMalformed))
		})

		It("returns ErrTokenMalformed if token was encrypted with a different key", func() {
			otherCodec := crypto.NewTokenCodec(paseto.NewV4SymmetricKey())
			tokenStr, err := otherCodec.Encode(validPayload, time.Minute)
			Expect(err).NotTo(HaveOccurred())

			var decoded testPayload
			err = codec.Decode(tokenStr, &decoded)

			Expect(err).To(MatchError(crypto.ErrTokenMalformed))
		})

		It("returns ErrTokenMalformed if token is missing the 'payload' claim", func() {
			token := paseto.NewToken()
			token.SetExpiration(time.Now().Add(time.Minute))
			tokenStr := token.V4Encrypt(pasetoKey, nil)

			var decoded testPayload
			err := codec.Decode(tokenStr, &decoded)

			Expect(err).To(MatchError(crypto.ErrTokenMalformed))
		})

		It("returns error if payload cannot be unmarshaled into target type", func() {
			tokenStr, err := codec.Encode(validPayload, time.Minute)
			Expect(err).NotTo(HaveOccurred())

			var incompatibleTarget int
			err = codec.Decode(tokenStr, &incompatibleTarget)

			Expect(err).To(HaveOccurred())
			Expect(err).NotTo(MatchError(crypto.ErrTokenMalformed))
			Expect(err).NotTo(MatchError(crypto.ErrTokenExpired))
		})
	})

	Describe("String Redaction", func() {
		It("redacts privateKey", func() {
			Expect(codec.String()).To(Equal("TokenCodec{privateKey: ***REDACTED***}"))

			var nilCodec *crypto.TokenCodec
			Expect(nilCodec.String()).To(Equal("<nil>"))
		})
	})
})
