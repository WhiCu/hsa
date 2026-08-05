package webauthnadapter_test

import (
	"errors"
	"time"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	webauthnadapter "github.com/whicu/hsa/internal/infrastructure/auth/webauthn"
	"github.com/whicu/hsa/internal/infrastructure/auth/webauthn/mocks"
	"github.com/whicu/hsa/pkg/logger"
)

const testChallengeToken = "test-challenge-token"

func newTestWebAuthn(t interface{ Helper() }) *gowebauthn.WebAuthn {
	t.Helper()
	wa, err := gowebauthn.New(&gowebauthn.Config{
		RPDisplayName: "Test App",
		RPID:          "localhost",
		RPOrigins:     []string{"http://localhost:8080"},
	})
	Expect(err).NotTo(HaveOccurred())
	return wa
}

var _ = Describe("Registrator", func() {
	var (
		wa       *gowebauthn.WebAuthn
		codec    *mocks.ChallengeCodec
		reg      *webauthnadapter.Registrator
		ttl      time.Duration
		userID   uuid.UUID
		inviteID uuid.UUID
	)

	BeforeEach(func() {
		wa = newTestWebAuthn(GinkgoT())
		codec = mocks.NewChallengeCodec(GinkgoT())
		ttl = time.Minute
		reg = webauthnadapter.NewRegistrator(logger.NewNOPSlog(), wa, codec, ttl)

		userID = uuid.New()
		inviteID = uuid.New()
	})

	Describe("Begin", func() {
		It("should successfully create registration options and an encoded challenge token", func(ctx SpecContext) {
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

		It("should fail if challenge codec encoding returns an error", func(ctx SpecContext) {
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

		It("should fail if webauthn config is invalid", func(ctx SpecContext) {
			invalidWA, err := gowebauthn.New(&gowebauthn.Config{
				RPDisplayName: "",
				RPID:          "",
			})
			if err != nil {
				Skip("invalid webauthn config is now rejected at construction time")
			}

			regInvalid := webauthnadapter.NewRegistrator(logger.NewNOPSlog(), invalidWA, codec, ttl)

			token, optsJSON, err := regInvalid.Begin(ctx, userID, inviteID)

			Expect(err).To(HaveOccurred())
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

			res, err := reg.Finish(ctx, testChallengeToken, []byte("{}"))

			Expect(err).To(MatchError(webauthnadapter.ErrChallengeExpired))
			Expect(res.UserID).To(Equal(uuid.Nil))
		})

		It("should return error when raw credential creation response is invalid", func(ctx SpecContext) {
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

		// It("should return error when WebAuthn credential creation verification fails", func(ctx SpecContext) {
		// 	codec.EXPECT().
		// 		Decode(testChallengeToken, mock.Anything).
		// 		Return(nil).
		// 		Once()

		// 	parsed, err := protocol.ParseCredentialCreationResponseBytes([]byte(`{
		// 		"id": "dGVzdC1jcmVkLWlk",
		// 		"rawId": "dGVzdC1jcmVkLWlk",
		// 		"type": "public-key",
		// 		"response": {
		// 			"clientDataJSON": "eyJ0eXBlIjoid2ViYXV0aG4uY3JlYXRlIiwiY2hhbGxlbmdlIjoidGVzdCIsIm9yaWdpbiI6Imh0dHA6Ly9sb2NhbGhvc3Q6ODA4MCJ9",
		// 			"attestationObject": "o2NmbXRub25lZ2F0dFN0bXSAaGF1dGhEYXRhWCVAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"
		// 		}
		// 	}`))
		// 	Expect(err).NotTo(HaveOccurred())

		// 	rawBytes, err := json.Marshal(parsed.Raw)
		// 	Expect(err).NotTo(HaveOccurred())

		// 	res, err := reg.Finish(ctx, testChallengeToken, rawBytes)

		// 	Expect(err).To(HaveOccurred())
		// 	Expect(res.UserID).To(Equal(uuid.Nil))
		// })
	})
})
