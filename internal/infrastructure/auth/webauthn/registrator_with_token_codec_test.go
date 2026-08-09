package webauthnadapter_test

import (
	"time"

	"aidanwoods.dev/go-paseto"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	webauthnadapter "github.com/whicu/hsa/internal/infrastructure/auth/webauthn"
	"github.com/whicu/hsa/internal/infrastructure/crypto"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("RegistratorWithTokenCodec", func() {
	var (
		wa         *gowebauthn.WebAuthn
		tokenCodec *crypto.TokenCodec
		reg        *webauthnadapter.Registrator
		ttl        time.Duration
	)

	BeforeEach(func() {
		wa = newTestWebAuthn(GinkgoT())
		pasetoKey := paseto.NewV4SymmetricKey()
		tokenCodec = crypto.NewTokenCodec(pasetoKey)
		ttl = 5 * time.Minute
		reg = webauthnadapter.NewRegistrator(logger.NewNOPSlog(), wa, tokenCodec, ttl)
	})

	It("should successfully execute full Begin flow and generate a valid decryptable PASETO token", func(ctx SpecContext) {
		userID := uuid.New()
		inviteID := uuid.New()

		tokenStr, optsJSON, err := reg.Begin(ctx, userID, inviteID)

		Expect(err).NotTo(HaveOccurred())
		Expect(tokenStr).NotTo(BeEmpty())
		Expect(optsJSON).NotTo(BeNil())

		var decoded map[string]any
		err = tokenCodec.Decode(tokenStr, &decoded)
		Expect(err).NotTo(HaveOccurred())
		Expect(decoded).To(HaveKey("session_data"))
		Expect(decoded).To(HaveKey("invite_id"))
		Expect(decoded).To(HaveKey("user_id"))
	})

	It("should reject Finish operation when PASETO token is expired", func(ctx SpecContext) {
		expiredReg := webauthnadapter.NewRegistrator(logger.NewNOPSlog(), wa, tokenCodec, -time.Minute)

		userID := uuid.New()
		inviteID := uuid.New()

		expiredToken, _, err := expiredReg.Begin(ctx, userID, inviteID)
		Expect(err).NotTo(HaveOccurred())

		res, err := reg.Finish(ctx, expiredToken, []byte("{}"))

		Expect(err).To(MatchError(webauthnadapter.ErrChallengeExpired))
		Expect(res.UserID).To(Equal(uuid.Nil))
	})
	It("should reject Finish operation when token is tampered or signed with a different key", func(ctx SpecContext) {
		otherCodec := crypto.NewTokenCodec(paseto.NewV4SymmetricKey())
		otherReg := webauthnadapter.NewRegistrator(logger.NewNOPSlog(), wa, otherCodec, ttl)

		userID := uuid.New()
		inviteID := uuid.New()

		tamperedToken, _, err := otherReg.Begin(ctx, userID, inviteID)
		Expect(err).NotTo(HaveOccurred())

		res, err := reg.Finish(ctx, tamperedToken, []byte("{}"))

		Expect(err).To(MatchError(webauthnadapter.ErrChallengeExpired))
		Expect(res.UserID).To(Equal(uuid.Nil))
	})
})
