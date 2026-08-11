package crypto_test

import (
	"time"

	"aidanwoods.dev/go-paseto"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/whicu/hsa/internal/infrastructure/crypto"
)

type testPayload struct {
	ID   string `json:"id"`
	Data string `json:"data"`
}

const (
	testPayloadID   = "challenge-123"
	testPayloadData = "user-session-payload"
)

var _ = Describe("TokenCodec", func() {
	var (
		codec        *crypto.TokenCodec
		pasetoKey    paseto.V4SymmetricKey
		validPayload testPayload
	)

	BeforeEach(func() {
		pasetoKey = paseto.NewV4SymmetricKey()
		codec = crypto.NewTokenCodec(pasetoKey)
		validPayload = testPayload{
			ID:   testPayloadID,
			Data: testPayloadData,
		}
	})

	Describe("Encode", func() {
		It("should successfully encode a valid payload", func() {
			tokenStr, err := codec.Encode(validPayload, time.Minute)

			Expect(err).NotTo(HaveOccurred())
			Expect(tokenStr).NotTo(BeEmpty())
		})

		It("should fail encoding when payload cannot be marshaled to JSON", func() {
			unmarshalablePayload := make(chan int)

			tokenStr, err := codec.Encode(unmarshalablePayload, time.Minute)

			Expect(err).To(HaveOccurred())
			Expect(tokenStr).To(BeEmpty())
		})
	})

	Describe("Decode", func() {
		It("should successfully decode a valid token", func() {
			tokenStr, err := codec.Encode(validPayload, time.Minute)
			Expect(err).NotTo(HaveOccurred())

			var decoded testPayload
			err = codec.Decode(tokenStr, &decoded)

			Expect(err).NotTo(HaveOccurred())
			Expect(decoded).To(Equal(validPayload))
		})

		It("should return ErrTokenExpired when decoding an expired token", func() {
			tokenStr, err := codec.Encode(validPayload, -time.Minute)
			Expect(err).NotTo(HaveOccurred())

			var decoded testPayload
			err = codec.Decode(tokenStr, &decoded)

			Expect(err).To(MatchError(crypto.ErrTokenExpired))
		})

		It("should return ErrTokenMalformed for an invalid token string", func() {
			var decoded testPayload
			err := codec.Decode("invalid-paseto-token-string", &decoded)

			Expect(err).To(MatchError(crypto.ErrTokenMalformed))
		})

		It("should return ErrTokenMalformed if token was encrypted with a different key", func() {
			otherCodec := crypto.NewTokenCodec(paseto.NewV4SymmetricKey())
			tokenStr, err := otherCodec.Encode(validPayload, time.Minute)
			Expect(err).NotTo(HaveOccurred())

			var decoded testPayload
			err = codec.Decode(tokenStr, &decoded)

			Expect(err).To(MatchError(crypto.ErrTokenMalformed))
		})

		It("should return ErrTokenMalformed if token is missing the 'payload' claim", func() {
			token := paseto.NewToken()
			token.SetExpiration(time.Now().Add(time.Minute))
			tokenStr := token.V4Encrypt(pasetoKey, nil)

			var decoded testPayload
			err := codec.Decode(tokenStr, &decoded)

			Expect(err).To(MatchError(crypto.ErrTokenMalformed))
		})

		It("should return error if payload cannot be unmarshaled into output struct", func() {
			tokenStr, err := codec.Encode(validPayload, time.Minute)
			Expect(err).NotTo(HaveOccurred())

			var incompatibleTarget int
			err = codec.Decode(tokenStr, &incompatibleTarget)

			Expect(err).To(HaveOccurred())
			Expect(err).NotTo(MatchError(crypto.ErrTokenMalformed))
			Expect(err).NotTo(MatchError(crypto.ErrTokenExpired))
		})
	})

	Describe("String", func() {
		It("should redact privateKey", func() {
			Expect(codec.String()).To(Equal("TokenCodec{privateKey: ***REDACTED***}"))

			var nilCodec *crypto.TokenCodec
			Expect(nilCodec.String()).To(Equal("<nil>"))
		})
	})
})
