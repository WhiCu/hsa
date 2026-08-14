package webauthnadapter

import (
	"time"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"
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
		reg      *Registrator
		ttl      time.Duration
		userID   uuid.UUID
		inviteID uuid.UUID
	)

	BeforeEach(func() {
		wa = newTestWebAuthn(GinkgoT())
		codec = mocks.NewChallengeCodec(GinkgoT())
		ttl = time.Minute
		reg = NewRegistrator(logger.NewNOPSlog(), wa, codec, ttl)

		userID = uuid.New()
		inviteID = uuid.New()
	})
	It("should handle payload with unexpected user ID during Finish", func(ctx SpecContext) {

		codec.EXPECT().
			Decode(testChallengeToken, mock.Anything).
			Run(func(_ string, out any) {
				if p, ok := out.(*challengePayload); ok {
					p.UserID = userID
					p.InviteID = inviteID
				}
			}).
			Return(nil).
			Once()

		invalidResponse := []byte("invalid-json-response")
		res, err := reg.Finish(ctx, testChallengeToken, invalidResponse)

		Expect(err).To(HaveOccurred())
		Expect(res.UserID).To(Equal(uuid.Nil))
	})

	It("should stringify challengePayload redacting SessionData", func() {
		payload := challengePayload{
			SessionData: gowebauthn.SessionData{
				Challenge: "super-secret-challenge",
				UserID:    userID[:],
			},
			InviteID: inviteID,
			UserID:   userID,
		}

		str := payload.String()
		Expect(str).To(ContainSubstring("***REDACTED***"))
		Expect(str).NotTo(ContainSubstring("super-secret-challenge"))
		Expect(str).To(ContainSubstring("InviteID: " + inviteID.String()))
		Expect(str).To(ContainSubstring("UserID: " + userID.String()))
	})
})
