package webauthnadapter_test

import (
	"errors"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/mock"

	webauthnadapter "github.com/whicu/hsa/internal/infrastructure/auth/webauthn"
	"github.com/whicu/hsa/internal/infrastructure/auth/webauthn/mocks"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("Authenticator", func() {
	var (
		injector  do.Injector
		codec     *mocks.ChallengeCodec
		credsProv *mocks.CredentialsProvider
		auth      *webauthnadapter.Authenticator
		ttl       time.Duration
	)

	BeforeEach(func() {
		codec = mocks.NewChallengeCodec(GinkgoT())
		credsProv = mocks.NewCredentialsProvider(GinkgoT())
		ttl = testConfig.ChallengeTTL

		injector = do.New(webauthnadapter.Package)

		do.OverrideValue(injector, logger.NewNOPSlog())
		do.OverrideValue(injector, testConfig)
		do.OverrideValue[webauthnadapter.ChallengeCodec](injector, codec)
		do.OverrideValue[webauthnadapter.CredentialsProvider](injector, credsProv)

		var err error
		auth, err = do.Invoke[*webauthnadapter.Authenticator](injector)
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("Begin", func() {
		It("successfully creates discoverable login options and an encoded challenge token", func(ctx SpecContext) {
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

		It("fails when challenge codec encoding returns an error", func(ctx SpecContext) {
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
		It("returns ErrChallengeExpired when challenge token decoding fails", func(ctx SpecContext) {
			decodeErr := errors.New("decode failed")
			codec.EXPECT().
				Decode(testChallengeToken, mock.Anything).
				Return(decodeErr).
				Once()

			res, err := auth.Finish(ctx, testChallengeToken, []byte("{}"))

			Expect(err).To(MatchError(webauthnadapter.ErrChallengeExpired))
			Expect(res.UserID).To(Equal(uuid.Nil))
		})

		It("returns error when raw credential request response payload is invalid", func(ctx SpecContext) {
			codec.EXPECT().
				Decode(testChallengeToken, mock.Anything).
				Return(nil).
				Once()

			invalidResponse := []byte("invalid-json-response")

			res, err := auth.Finish(ctx, testChallengeToken, invalidResponse)

			Expect(err).To(HaveOccurred())
			Expect(err).NotTo(MatchError(webauthnadapter.ErrChallengeExpired))
			Expect(res.UserID).To(Equal(uuid.Nil))
		})

		It("fails when ParseCredentialRequestResponseBytes fails", func(ctx SpecContext) {
			codec.EXPECT().Decode(testChallengeToken, mock.Anything).Return(nil).Once()
			invalidResponse := []byte(`{"id": "not-valid", "type": "public-key"}`)
			res, err := auth.Finish(ctx, testChallengeToken, invalidResponse)
			Expect(err).To(HaveOccurred())
			Expect(err).NotTo(MatchError(webauthnadapter.ErrChallengeExpired))
			Expect(res.UserID).To(Equal(uuid.Nil))
		})
	})
})
