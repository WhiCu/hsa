package webauthnadapter_test

import (
	"errors"
	"time"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	webauthnadapter "github.com/whicu/hsa/internal/infrastructure/auth/webauthn"
	"github.com/whicu/hsa/internal/infrastructure/auth/webauthn/mocks"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("Authenticator", func() {
	var (
		wa        *gowebauthn.WebAuthn
		codec     *mocks.ChallengeCodec
		credsProv *mocks.CredentialsProvider
		auth      *webauthnadapter.Authenticator
		ttl       time.Duration
	)

	BeforeEach(func() {
		wa = newTestWebAuthn(GinkgoT())
		codec = mocks.NewChallengeCodec(GinkgoT())
		credsProv = mocks.NewCredentialsProvider(GinkgoT())
		ttl = time.Minute
		auth = webauthnadapter.NewAuthenticator(logger.NewNOPSlog(), wa, codec, credsProv, ttl)
	})

	Describe("Begin", func() {
		It("should successfully create login options and an encoded challenge token", func(ctx SpecContext) {
			codec.EXPECT().
				Encode(mock.Anything, ttl).
				Return(testChallengeToken, nil).
				Once()

			token, optsJSON, err := auth.Begin(ctx)

			Expect(err).NotTo(HaveOccurred())
			Expect(token).To(Equal(testChallengeToken))
			Expect(optsJSON).NotTo(BeEmpty())
			Expect(string(optsJSON)).To(ContainSubstring("publicKey"))
		})

		It("should fail if challenge codec encoding returns an error", func(ctx SpecContext) {
			encodeErr := errors.New("encode failed")
			codec.EXPECT().
				Encode(mock.Anything, ttl).
				Return("", encodeErr).
				Once()

			token, optsJSON, err := auth.Begin(ctx)

			Expect(err).To(MatchError(encodeErr))
			Expect(token).To(BeEmpty())
			Expect(optsJSON).To(BeNil())
		})
	})

	Describe("Finish", func() {
		It("should return ErrChallengeExpired if token decoding fails", func(ctx SpecContext) {
			decodeErr := errors.New("decode failed")
			codec.EXPECT().
				Decode(testChallengeToken, mock.Anything).
				Return(decodeErr).
				Once()

			res, err := auth.Finish(ctx, testChallengeToken, []byte("{}"))

			Expect(err).To(MatchError(webauthnadapter.ErrChallengeExpired))
			Expect(res.UserID.String()).To(Equal("00000000-0000-0000-0000-000000000000"))
		})

		It("should return error when raw credential request response is invalid", func(ctx SpecContext) {
			codec.EXPECT().
				Decode(testChallengeToken, mock.Anything).
				Return(nil).
				Once()

			invalidResponse := []byte("invalid-json-response")

			res, err := auth.Finish(ctx, testChallengeToken, invalidResponse)

			Expect(err).To(HaveOccurred())
			Expect(err).NotTo(MatchError(webauthnadapter.ErrChallengeExpired))
			Expect(res.UserID.String()).To(Equal("00000000-0000-0000-0000-000000000000"))
		})
	})
})
