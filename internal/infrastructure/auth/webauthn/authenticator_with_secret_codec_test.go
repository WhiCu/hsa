package webauthnadapter_test

import (
	"time"

	"aidanwoods.dev/go-paseto"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	webauthnadapter "github.com/whicu/hsa/internal/infrastructure/auth/webauthn"
	"github.com/whicu/hsa/internal/infrastructure/auth/webauthn/mocks"
	"github.com/whicu/hsa/internal/infrastructure/crypto"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("AuthenticatorWithSecretCodec", func() {
	var (
		wa          *gowebauthn.WebAuthn
		secretCodec *crypto.SecretCodec
		credsProv   *mocks.CredentialsProvider
		auth        *webauthnadapter.Authenticator
		ttl         time.Duration
	)

	BeforeEach(func() {
		wa = newTestWebAuthn(GinkgoT())
		pasetoKey := paseto.NewV4SymmetricKey()
		secretCodec = crypto.NewSecretCodec(pasetoKey)
		credsProv = mocks.NewCredentialsProvider(GinkgoT())
		ttl = 5 * time.Minute
		auth = webauthnadapter.NewAuthenticator(logger.NewNOPSlog(), wa, secretCodec, credsProv, ttl)
	})

	It("should successfully execute full Begin flow and generate a valid decryptable PASETO token", func(ctx SpecContext) {
		tokenStr, optsJSON, err := auth.Begin(ctx)

		Expect(err).NotTo(HaveOccurred())
		Expect(tokenStr).NotTo(BeEmpty())
		Expect(optsJSON).NotTo(BeNil())

		var decoded map[string]any
		err = secretCodec.Decode(tokenStr, &decoded)
		Expect(err).NotTo(HaveOccurred())
		Expect(decoded).To(HaveKey("session_data"))
	})

	It("should reject Finish operation when PASETO token is expired", func(ctx SpecContext) {
		expiredAuth := webauthnadapter.NewAuthenticator(logger.NewNOPSlog(), wa, secretCodec, credsProv, -time.Minute)

		expiredToken, _, err := expiredAuth.Begin(ctx)
		Expect(err).NotTo(HaveOccurred())

		res, err := auth.Finish(ctx, expiredToken, []byte("{}"))

		Expect(err).To(MatchError(webauthnadapter.ErrChallengeExpired))
		Expect(res.UserID.String()).To(Equal("00000000-0000-0000-0000-000000000000"))
	})

	It("should reject Finish operation when token is tampered or signed with a different key", func(ctx SpecContext) {
		otherCodec := crypto.NewSecretCodec(paseto.NewV4SymmetricKey())
		otherAuth := webauthnadapter.NewAuthenticator(logger.NewNOPSlog(), wa, otherCodec, credsProv, ttl)

		tamperedToken, _, err := otherAuth.Begin(ctx)
		Expect(err).NotTo(HaveOccurred())

		res, err := auth.Finish(ctx, tamperedToken, []byte("{}"))

		Expect(err).To(MatchError(webauthnadapter.ErrChallengeExpired))
		Expect(res.UserID.String()).To(Equal("00000000-0000-0000-0000-000000000000"))
	})
})
