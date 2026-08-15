package webauthnadapter_test

import (
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"

	webauthnadapter "github.com/whicu/hsa/internal/infrastructure/auth/webauthn"
	"github.com/whicu/hsa/internal/infrastructure/crypto"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("Registrator with Real TokenCodec (PASETO)", func() {
	var (
		injector   do.Injector
		tokenCodec *crypto.TokenCodec
		reg        *webauthnadapter.Registrator
		ttl        time.Duration
	)

	BeforeEach(func() {
		pasetoKey := paseto.NewV4SymmetricKey()
		tokenCodec = crypto.NewTokenCodec(pasetoKey)
		ttl = 5 * time.Minute

		cfg := testConfig
		cfg.ChallengeTTL = ttl

		injector = do.New(webauthnadapter.Package)

		do.OverrideValue(injector, logger.NewNOPSlog())
		do.OverrideValue(injector, cfg)
		do.OverrideValue[webauthnadapter.ChallengeCodec](injector, tokenCodec)

		var err error
		reg, err = do.Invoke[*webauthnadapter.Registrator](injector)
		Expect(err).NotTo(HaveOccurred())
	})

	It("successfully executes full Begin flow and generates a valid decryptable PASETO token", func(ctx SpecContext) {
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

	It("rejects Finish operation when PASETO token is expired", func(ctx SpecContext) {
		expiredCfg := testConfig
		expiredCfg.ChallengeTTL = -time.Minute

		expiredInjector := do.New(webauthnadapter.Package)
		do.OverrideValue(expiredInjector, logger.NewNOPSlog())
		do.OverrideValue(expiredInjector, expiredCfg)
		do.OverrideValue[webauthnadapter.ChallengeCodec](expiredInjector, tokenCodec)

		expiredReg, err := do.Invoke[*webauthnadapter.Registrator](expiredInjector)
		Expect(err).NotTo(HaveOccurred())

		userID := uuid.New()
		inviteID := uuid.New()

		expiredToken, _, err := expiredReg.Begin(ctx, userID, inviteID)
		Expect(err).NotTo(HaveOccurred())

		res, err := reg.Finish(ctx, expiredToken, []byte("{}"))

		Expect(err).To(MatchError(webauthnadapter.ErrChallengeExpired))
		Expect(res.UserID).To(Equal(uuid.Nil))
	})

	It("rejects Finish operation when token is tampered or signed with a different key", func(ctx SpecContext) {
		otherCodec := crypto.NewTokenCodec(paseto.NewV4SymmetricKey())

		otherInjector := do.New(webauthnadapter.Package)
		do.OverrideValue(otherInjector, logger.NewNOPSlog())
		do.OverrideValue(otherInjector, testConfig)
		do.OverrideValue[webauthnadapter.ChallengeCodec](otherInjector, otherCodec)

		otherReg, err := do.Invoke[*webauthnadapter.Registrator](otherInjector)
		Expect(err).NotTo(HaveOccurred())

		userID := uuid.New()
		inviteID := uuid.New()

		tamperedToken, _, err := otherReg.Begin(ctx, userID, inviteID)
		Expect(err).NotTo(HaveOccurred())

		res, err := reg.Finish(ctx, tamperedToken, []byte("{}"))

		Expect(err).To(MatchError(webauthnadapter.ErrChallengeExpired))
		Expect(res.UserID).To(Equal(uuid.Nil))
	})
})
