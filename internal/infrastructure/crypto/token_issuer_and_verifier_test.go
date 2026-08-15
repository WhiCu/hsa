package crypto_test

import (
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"

	"github.com/whicu/hsa/internal/domain/user"
	"github.com/whicu/hsa/internal/infrastructure/crypto"
)

var _ = Describe("AccessTokenIssuer & AccessTokenVerifier", func() {
	var (
		injector do.Injector
		issuer   *crypto.AccessTokenIssuer
		verifier *crypto.AccessTokenVerifier
		testUser *user.User
		userID   user.UserID
	)

	BeforeEach(func() {

		injector = do.New(crypto.Package)
		do.OverrideValue(injector, testConfig)

		var err error
		issuer, err = do.Invoke[*crypto.AccessTokenIssuer](injector)
		Expect(err).NotTo(HaveOccurred())

		verifier, err = do.Invoke[*crypto.AccessTokenVerifier](injector)
		Expect(err).NotTo(HaveOccurred())

		userID = uuid.New()
		testUser, err = user.NewRoot(userID, time.Now())
		Expect(err).NotTo(HaveOccurred())
	})

	Describe("IssueAccessToken & Verify", func() {
		It("successfully issues and verifies a signed PASETO v4 public access token", func() {
			tokenStr, err := issuer.IssueAccessToken(testUser.ID(), 15*time.Minute)

			Expect(err).NotTo(HaveOccurred())
			Expect(tokenStr).NotTo(BeEmpty())
			Expect(tokenStr).To(HavePrefix("v4.public."))

			verifiedUserID, err := verifier.Verify(tokenStr)
			Expect(err).NotTo(HaveOccurred())
			Expect(verifiedUserID).To(Equal(testUser.ID()))
		})

		It("returns ErrTokenExpired when verifying an expired access token", func() {
			tokenStr, err := issuer.IssueAccessToken(testUser.ID(), -time.Minute)
			Expect(err).NotTo(HaveOccurred())

			_, err = verifier.Verify(tokenStr)
			Expect(err).To(MatchError(crypto.ErrTokenExpired))
		})

		It("returns ErrTokenMalformed when token was signed by a different private key", func() {
			otherKey := paseto.NewV4AsymmetricSecretKey()
			otherIssuer := crypto.NewAccessTokenIssuer(otherKey)

			tokenStr, err := otherIssuer.IssueAccessToken(testUser.ID(), 15*time.Minute)
			Expect(err).NotTo(HaveOccurred())

			_, err = verifier.Verify(tokenStr)
			Expect(err).To(MatchError(crypto.ErrTokenMalformed))
		})

		It("returns ErrTokenMalformed for a corrupted token string", func() {
			_, err := verifier.Verify("v4.public.invalid-signature-payload")
			Expect(err).To(MatchError(crypto.ErrTokenMalformed))
		})

		It("returns ErrTokenMalformed if token is missing the user_id claim", func() {
			token := paseto.NewToken()
			token.SetExpiration(time.Now().Add(time.Minute))

			asymKey := paseto.NewV4AsymmetricSecretKey()
			tokenStr := token.V4Sign(asymKey, nil)

			customVerifier := crypto.NewAccessTokenVerifier(asymKey.Public())
			_, err := customVerifier.Verify(tokenStr)

			Expect(err).To(MatchError(crypto.ErrTokenMalformed))
		})
	})

	Describe("Automatic Public Key Derivation", func() {
		It("verifies tokens when PublicKey is omitted in Config", func() {
			cfg := testConfig
			cfg.PASETO.PublicKey = "" // Очищаем явный публичный ключ

			customInjector := do.New(crypto.Package)
			do.OverrideValue(customInjector, cfg)

			customIssuer, err := do.Invoke[*crypto.AccessTokenIssuer](customInjector)
			Expect(err).NotTo(HaveOccurred())

			customVerifier, err := do.Invoke[*crypto.AccessTokenVerifier](customInjector)
			Expect(err).NotTo(HaveOccurred())

			tokenStr, err := customIssuer.IssueAccessToken(userID, 5*time.Minute)
			Expect(err).NotTo(HaveOccurred())

			verifiedID, err := customVerifier.Verify(tokenStr)
			Expect(err).NotTo(HaveOccurred())
			Expect(verifiedID).To(Equal(userID))
		})
	})

	Describe("String Redaction", func() {
		It("redacts secretKey in AccessTokenIssuer", func() {
			Expect(issuer.String()).To(Equal("AccessTokenIssuer{secretKey: ***REDACTED***}"))

			var nilIssuer *crypto.AccessTokenIssuer
			Expect(nilIssuer.String()).To(Equal("<nil>"))
		})

		It("redacts publicKey in AccessTokenVerifier", func() {
			Expect(verifier.String()).To(Equal("AccessTokenVerifier{publicKey: ***REDACTED***}"))

			var nilVerifier *crypto.AccessTokenVerifier
			Expect(nilVerifier.String()).To(Equal("<nil>"))
		})
	})
})
