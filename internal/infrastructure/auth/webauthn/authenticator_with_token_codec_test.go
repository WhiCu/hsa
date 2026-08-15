package webauthnadapter_test

import (
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"

	webauthnadapter "github.com/whicu/hsa/internal/infrastructure/auth/webauthn"
	"github.com/whicu/hsa/internal/infrastructure/auth/webauthn/mocks"
	"github.com/whicu/hsa/internal/infrastructure/crypto"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("Authenticator with Real TokenCodec (PASETO)", func() {
	var (
		injector   do.Injector
		tokenCodec *crypto.TokenCodec
		credsProv  *mocks.CredentialsProvider
		auth       *webauthnadapter.Authenticator
		ttl        time.Duration
	)

	BeforeEach(func() {
		pasetoKey := paseto.NewV4SymmetricKey()
		tokenCodec = crypto.NewTokenCodec(pasetoKey)
		credsProv = mocks.NewCredentialsProvider(GinkgoT())
		ttl = 5 * time.Minute

		cfg := testConfig
		cfg.ChallengeTTL = ttl

		injector = do.New(webauthnadapter.Package)

		do.OverrideValue(injector, logger.NewNOPSlog())
		do.OverrideValue(injector, cfg)
		do.OverrideValue[webauthnadapter.ChallengeCodec](injector, tokenCodec)
		do.OverrideValue[webauthnadapter.CredentialsProvider](injector, credsProv)

		var err error
		auth, err = do.Invoke[*webauthnadapter.Authenticator](injector)
		Expect(err).NotTo(HaveOccurred())
	})

	It("successfully executes Begin flow and generates a valid decryptable PASETO token", func(ctx SpecContext) {
		tokenStr, optsJSON, err := auth.Begin(ctx)

		Expect(err).NotTo(HaveOccurred())
		Expect(tokenStr).NotTo(BeEmpty())
		Expect(optsJSON).NotTo(BeNil())

		var decoded map[string]any
		err = tokenCodec.Decode(tokenStr, &decoded)
		Expect(err).NotTo(HaveOccurred())
		Expect(decoded).To(HaveKey("session_data"))
	})

	It("rejects Finish operation when PASETO token is expired", func(ctx SpecContext) {
		expiredCfg := testConfig
		expiredCfg.ChallengeTTL = -time.Minute

		expiredInjector := do.New(webauthnadapter.Package)
		do.OverrideValue(expiredInjector, logger.NewNOPSlog())
		do.OverrideValue(expiredInjector, expiredCfg)
		do.OverrideValue[webauthnadapter.ChallengeCodec](expiredInjector, tokenCodec)
		do.OverrideValue[webauthnadapter.CredentialsProvider](expiredInjector, credsProv)

		expiredAuth, err := do.Invoke[*webauthnadapter.Authenticator](expiredInjector)
		Expect(err).NotTo(HaveOccurred())

		expiredToken, _, err := expiredAuth.Begin(ctx)
		Expect(err).NotTo(HaveOccurred())

		res, err := auth.Finish(ctx, expiredToken, []byte("{}"))

		Expect(err).To(MatchError(webauthnadapter.ErrChallengeExpired))
		Expect(res.UserID).To(Equal(uuid.Nil))
	})

	It("rejects Finish operation when token is tampered or signed with a different key", func(ctx SpecContext) {
		otherCodec := crypto.NewTokenCodec(paseto.NewV4SymmetricKey())

		otherInjector := do.New(webauthnadapter.Package)
		do.OverrideValue(otherInjector, logger.NewNOPSlog())
		do.OverrideValue(otherInjector, testConfig)
		do.OverrideValue[webauthnadapter.ChallengeCodec](otherInjector, otherCodec)
		do.OverrideValue[webauthnadapter.CredentialsProvider](otherInjector, credsProv)

		otherAuth, err := do.Invoke[*webauthnadapter.Authenticator](otherInjector)
		Expect(err).NotTo(HaveOccurred())

		tamperedToken, _, err := otherAuth.Begin(ctx)
		Expect(err).NotTo(HaveOccurred())

		res, err := auth.Finish(ctx, tamperedToken, []byte("{}"))

		Expect(err).To(MatchError(webauthnadapter.ErrChallengeExpired))
		Expect(res.UserID).To(Equal(uuid.Nil))
	})
})
