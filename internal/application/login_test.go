package application_test

import (
	"errors"
	"net/netip"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/mock"

	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/credential"
	"github.com/whicu/hsa/internal/domain/key"
	"github.com/whicu/hsa/internal/domain/user"
)

var _ = Describe("Login UseCase", func() {
	var (
		injector do.Injector
		m        *Mocks

		beginUC  *application.BeginLogin
		finishUC *application.Login

		fixedNow       time.Time
		challengeToken string
		authResp       []byte
		externalID     []byte
		userID         user.UserID
		credID         credential.CredentialID
		testUser       *user.User
	)

	BeforeEach(func() {
		fixedNow = time.Date(2026, time.August, 4, 12, 0, 0, 0, time.UTC)
		challengeToken = "challenge-token"
		authResp = []byte("valid-webauthn-response")
		externalID = []byte("credential-external-id")
		userID = uuid.New()
		credID = uuid.New()

		// Тестовый пользователь, который будет найден при выпуске сессии
		testUser = user.Reconstruct(userID, user.Member, nil, fixedNow)

		injector = do.New(application.Package)
		m = MockPackage(injector, GinkgoT())

		var err error
		beginUC, err = do.Invoke[*application.BeginLogin](injector)
		Expect(err).ToNot(HaveOccurred())

		finishUC, err = do.Invoke[*application.Login](injector)
		Expect(err).ToNot(HaveOccurred())
	})

	// --- Helper Functions ---

	newInput := func() application.LoginInput {
		return application.LoginInput{
			ChallengeToken:         challengeToken,
			AuthenticationResponse: authResp,
			DeviceInfo:             "Chrome on macOS",
			IPAddress:              netip.MustParseAddr("192.168.1.100"),
		}
	}

	newAuthResult := func(newSignCount uint32) application.AuthenticationResult {
		return application.AuthenticationResult{
			UserID:       userID,
			ExternalID:   externalID,
			NewSignCount: newSignCount,
		}
	}

	newCredential := func(initialSignCount uint32) *credential.Credential {
		cred, err := credential.New(credID, externalID, userID, []byte("pub-key"), []string{"usb"}, fixedNow)
		Expect(err).ToNot(HaveOccurred())
		cred.SetSignCount(initialSignCount)
		return cred
	}

	newTestWrappedKeys := func(cID uuid.UUID) []*key.WrappedKey {
		k1 := key.Reconstruct(uuid.New(), userID, cID, key.ScopeMain, []byte("main-dek-encrypted"), "AES-256-GCM-KW", fixedNow)
		k2 := key.Reconstruct(uuid.New(), userID, cID, key.ScopeDecoy, []byte("decoy-dek-encrypted"), "AES-256-GCM-KW", fixedNow)
		return []*key.WrappedKey{k1, k2}
	}

	expectSessionIssueOK := func(expectedAccess, expectedRefresh string) {
		// Ожидания для SessionIssuer, который вызывается внутри Login
		m.TokenGenerator.EXPECT().GenerateToken(32).Return(expectedRefresh, "hash", nil).Once()
		m.IDGenerator.EXPECT().NewID().Return(uuid.New()).Once()
		m.RefreshTokenSaver.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

		m.UserFinderByID.EXPECT().FindByID(mock.Anything, userID).Return(testUser, nil).Once()
		m.TokenIssuer.EXPECT().IssueAccessToken(userID, testUser.Role(), 15*time.Minute).Return(expectedAccess, nil).Once()
	}

	// --- BeginLogin UseCase ---

	Describe("BeginLogin", func() {
		It("successfully returns challengeToken and requestOptions", func(ctx SpecContext) {
			expectedOpts := []byte("req-opts")
			m.Authenticator.EXPECT().Begin(ctx).Return(challengeToken, expectedOpts, nil).Once()

			token, opts, err := beginUC.Execute(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(token).To(Equal(challengeToken))
			Expect(opts).To(Equal(expectedOpts))
		})

		It("fails when Authenticator.Begin returns an error", func(ctx SpecContext) {
			authErr := errors.New("failed to generate webauthn challenge")
			m.Authenticator.EXPECT().Begin(ctx).Return("", nil, authErr).Once()

			token, opts, err := beginUC.Execute(ctx)

			Expect(err).To(MatchError(authErr))
			Expect(token).To(BeEmpty())
			Expect(opts).To(BeNil())
		})
	})

	// --- FinishLogin UseCase ---

	Describe("Login (Execute)", func() {
		It("successfully finishes login: updates sign count, retrieves wrapped keys and issues tokens", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				newSignCount := uint32(42)
				authResult := newAuthResult(newSignCount)
				m.Authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

				cred := newCredential(10)
				m.CredentialFinder.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()

				m.CredentialSaver.EXPECT().Save(ctx, mock.MatchedBy(func(c *credential.Credential) bool {
					return c.SignCount() == newSignCount && c.UserID() == userID
				})).Return(nil).Once()

				keys := newTestWrappedKeys(cred.ID())
				m.CredentialWrappedKeysFinder.EXPECT().FindByCredentialID(ctx, cred.ID()).Return(keys, nil).Once()

				expectedRefreshCode := "refresh-code"
				expectedAccessCode := "access-code"
				expectSessionIssueOK(expectedAccessCode, expectedRefreshCode)

				out, err := finishUC.Execute(ctx, newInput())

				Expect(err).ToNot(HaveOccurred())
				Expect(out).ToNot(BeNil())
				Expect(out.AccessToken).To(Equal(expectedAccessCode))
				Expect(out.RefreshToken).To(Equal(expectedRefreshCode))
				Expect(out.WrappedKeys).To(HaveLen(2))
				Expect(out.WrappedKeys[0]).To(Equal(application.WrappedKeyOutput{
					Scope:         key.ScopeMain,
					WrappedDEK:    []byte("main-dek-encrypted"),
					WrapAlgorithm: "AES-256-GCM-KW",
				}))
			})
		})

		It("fails when Authenticator.Finish returns error", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				authErr := errors.New("invalid signature or challenge expired")
				m.Authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(application.AuthenticationResult{}, authErr).Once()

				out, err := finishUC.Execute(ctx, newInput())

				Expect(err).To(MatchError(authErr))
				Expect(out).To(BeNil())
			})
		})

		It("returns ErrCredentialNotFound when external ID is missing in database", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				authResult := newAuthResult(42)
				m.Authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

				m.CredentialFinder.EXPECT().FindByExternalID(ctx, externalID).Return(nil, domain.ErrNotFound).Once()

				out, err := finishUC.Execute(ctx, newInput())

				Expect(err).To(MatchError(application.ErrCredentialNotFound))
				Expect(out).To(BeNil())
			})
		})

		It("returns ErrCredentialRevoked when credential is already revoked", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				authResult := newAuthResult(42)
				m.Authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

				cred := newCredential(10)
				_ = cred.Revoke(fixedNow)
				m.CredentialFinder.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()

				out, err := finishUC.Execute(ctx, newInput())

				Expect(err).To(MatchError(application.ErrCredentialRevoked))
				Expect(out).To(BeNil())
			})
		})

		It("detects clone via CloneWarning, revokes credential and sessions, and returns ErrCredentialCloneSuspected", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				authResult := newAuthResult(42)
				authResult.CloneWarning = true // Сигнал от аутентификатора
				m.Authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

				cred := newCredential(10)
				m.CredentialFinder.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()

				// Ожидаем сохранение отозванного креда
				m.CredentialSaver.EXPECT().Save(mock.Anything, mock.MatchedBy(func(c *credential.Credential) bool {
					return c.IsRevoked()
				})).Return(nil).Once()

				// Ожидаем вызов RevokeAllUserSessions
				m.ActiveSessionsFinder.EXPECT().FindActiveByUserIDs(mock.Anything, []user.UserID{userID}, mock.Anything).Return(nil, nil).Once()

				out, err := finishUC.Execute(ctx, newInput())

				Expect(err).To(MatchError(application.ErrCredentialCloneSuspected))
				Expect(out).To(BeNil())
			})
		})

		It("detects clone via SignCount regression, revokes credential and sessions, and returns ErrCredentialCloneSuspected", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				authResult := newAuthResult(5) // Значение 5 меньше текущего (10)
				m.Authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

				cred := newCredential(10)
				m.CredentialFinder.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()

				m.CredentialSaver.EXPECT().Save(mock.Anything, mock.MatchedBy(func(c *credential.Credential) bool {
					return c.IsRevoked()
				})).Return(nil).Once()

				m.ActiveSessionsFinder.EXPECT().FindActiveByUserIDs(mock.Anything, []user.UserID{userID}, mock.Anything).Return(nil, nil).Once()

				out, err := finishUC.Execute(ctx, newInput())

				Expect(err).To(MatchError(application.ErrCredentialCloneSuspected))
				Expect(out).To(BeNil())
			})
		})

		Context("Transaction Failures & Rollback Checks", func() {
			It("rolls back when CredentialSaver.Save fails on sign count update", func(ctx SpecContext) {
				synctest.Test(testT, func(_ *testing.T) {
					authResult := newAuthResult(42)
					m.Authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

					cred := newCredential(10)
					m.CredentialFinder.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()

					saveErr := errors.New("failed to update credential sign count")
					m.CredentialSaver.EXPECT().Save(mock.Anything, mock.Anything).Return(saveErr).Once()

					out, err := finishUC.Execute(ctx, newInput())

					Expect(err).To(MatchError(saveErr))
					Expect(out).To(BeNil())
				})
			})

			It("rolls back when wrappedKeys.FindByCredentialID fails", func(ctx SpecContext) {
				synctest.Test(testT, func(_ *testing.T) {
					authResult := newAuthResult(42)
					m.Authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

					cred := newCredential(10)
					m.CredentialFinder.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()
					m.CredentialSaver.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

					wkErr := errors.New("failed to query wrapped keys")
					m.CredentialWrappedKeysFinder.EXPECT().FindByCredentialID(mock.Anything, cred.ID()).Return(nil, wkErr).Once()

					out, err := finishUC.Execute(ctx, newInput())

					Expect(err).To(MatchError(wkErr))
					Expect(out).To(BeNil())
				})
			})

			It("rolls back when IssueAccessToken fails during SessionIssuer call", func(ctx SpecContext) {
				synctest.Test(testT, func(_ *testing.T) {
					authResult := newAuthResult(42)
					m.Authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

					cred := newCredential(10)
					m.CredentialFinder.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()
					m.CredentialSaver.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()
					m.CredentialWrappedKeysFinder.EXPECT().FindByCredentialID(mock.Anything, cred.ID()).Return(newTestWrappedKeys(cred.ID()), nil).Once()

					// Ломаем генерацию токена в SessionIssuer
					tokenErr := errors.New("entropy error on refresh token gen")
					m.TokenGenerator.EXPECT().GenerateToken(32).Return("", "", tokenErr).Once()

					out, err := finishUC.Execute(ctx, newInput())

					Expect(err).To(MatchError(tokenErr))
					Expect(out).To(BeNil())
				})
			})
		})
	})

	Context("Helper Data Structures", func() {
		It("WrappedKeyOutput String redaction", func() {
			wki := application.WrappedKeyOutput{
				Scope:         key.ScopeMain,
				WrappedDEK:    []byte("my-secret-dek-1234567890"), // 24 bytes
				WrapAlgorithm: "AES-256-GCM",
			}
			expected := "WrappedKeyOutput{Scope: 0, WrappedDEK: 24 bytes, WrapAlgorithm: AES-256-GCM}"
			Expect(wki.String()).To(Equal(expected))
		})

		It("LoginInput String redaction", func() {
			in := newInput()
			expected := "LoginInput{ChallengeToken: ***REDACTED***, AuthenticationResponse: ***REDACTED***, DeviceInfo: Chrome on macOS, IPAddress: 192.168.1.100}"
			Expect(in.String()).To(Equal(expected))
		})

		It("LoginOutput String redaction", func() {
			out := application.LoginOutput{
				AccessToken:  "access",
				RefreshToken: "refresh",
				WrappedKeys:  make([]application.WrappedKeyOutput, 3),
			}
			expected := "LoginOutput{AccessToken: ***REDACTED***, RefreshToken: ***REDACTED***, WrappedKeys: 3 records}"
			Expect(out.String()).To(Equal(expected))
		})
	})
})
