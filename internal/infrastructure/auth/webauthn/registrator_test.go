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

var _ = Describe("Registrator", func() {
	var (
		injector do.Injector
		codec    *mocks.ChallengeCodec
		reg      *webauthnadapter.Registrator
		ttl      time.Duration
		userID   uuid.UUID
		inviteID uuid.UUID
	)

	BeforeEach(func() {
		codec = mocks.NewChallengeCodec(GinkgoT())
		ttl = testConfig.ChallengeTTL

		injector = do.New(webauthnadapter.Package)

		do.OverrideValue(injector, logger.NewNOPSlog())
		do.OverrideValue(injector, testConfig)
		do.OverrideValue[webauthnadapter.ChallengeCodec](injector, codec)

		var err error
		reg, err = do.Invoke[*webauthnadapter.Registrator](injector)
		Expect(err).NotTo(HaveOccurred())

		userID = uuid.New()
		inviteID = uuid.New()
	})

	Describe("Begin", func() {
		It("successfully creates registration options and an encoded challenge token", func(ctx SpecContext) {
			codec.EXPECT().
				Encode(mock.Anything, ttl).
				Return(testChallengeToken, nil).
				Once()

			token, optsJSON, err := reg.Begin(ctx, userID, inviteID)

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

			token, optsJSON, err := reg.Begin(ctx, userID, inviteID)

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

			res, err := reg.Finish(ctx, testChallengeToken, []byte("{}"))

			Expect(err).To(MatchError(webauthnadapter.ErrChallengeExpired))
			Expect(res.UserID).To(Equal(uuid.Nil))
		})

		It("returns error when raw credential creation response payload is invalid", func(ctx SpecContext) {
			codec.EXPECT().
				Decode(testChallengeToken, mock.Anything).
				Return(nil).
				Once()

			invalidResponse := []byte("invalid-json-response")

			res, err := reg.Finish(ctx, testChallengeToken, invalidResponse)

			Expect(err).To(HaveOccurred())
			Expect(err).NotTo(MatchError(webauthnadapter.ErrChallengeExpired))
			Expect(res.UserID).To(Equal(uuid.Nil))
		})

		It("fails when ParseCredentialCreationResponseBytes fails", func(ctx SpecContext) {
			codec.EXPECT().Decode(testChallengeToken, mock.Anything).Return(nil).Once()
			invalidResponse := []byte(`{"id": "not-valid", "type": "public-key"}`)
			res, err := reg.Finish(ctx, testChallengeToken, invalidResponse)
			Expect(err).To(HaveOccurred())
			Expect(err).NotTo(MatchError(webauthnadapter.ErrChallengeExpired))
			Expect(res.UserID).To(Equal(uuid.Nil))
		})

		It("fails when session decoding succeeds but CreateCredential fails", func(ctx SpecContext) {
			codec.EXPECT().Decode(testChallengeToken, mock.Anything).Return(nil).Once()
			invalidResponse := []byte(`{
                "id": "abc",
                "rawId": "YWJj",
                "type": "public-key",
                "response": {
                    "clientDataJSON": "eyJjaGFsbGVuZ2UiOiAiY2hhbGxlbmdlIiwgIm9yaWdpbiI6ICJvcmlnaW4iLCAidHlwZSI6ICJ3ZWJhdXRobi5jcmVhdGUifQ==",
                    "attestationObject": "YXR0ZXN0YXRpb25PYmplY3Q="
                }
            }`)
			res, err := reg.Finish(ctx, testChallengeToken, invalidResponse)
			Expect(err).To(HaveOccurred())
			Expect(err).NotTo(MatchError(webauthnadapter.ErrChallengeExpired))
			Expect(res.UserID).To(Equal(uuid.Nil))
		})
	})
})
