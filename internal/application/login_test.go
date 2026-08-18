package application_test

import (
	"context"
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
	"github.com/whicu/hsa/internal/application/mocks"
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/credential"
	"github.com/whicu/hsa/internal/domain/key"
	"github.com/whicu/hsa/internal/domain/user"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("Login UseCase", func() {
	var (
		injector        do.Injector
		authenticator   *mocks.Authenticator
		credentials     *mocks.CredentialFinder
		credentialSaver *mocks.CredentialSaver
		wrappedKeys     *mocks.CredentialWrappedKeysFinder
		sessions        *mocks.RefreshTokenSaver
		refreshTokens   *mocks.TokenGenerator
		accessTokens    *mocks.TokenIssuer
		ids             *mocks.IDGenerator
		transactor      *mocks.Transactor
		revokeSessions  *mocks.ActiveSessionsFinder

		beginUC  *application.BeginLogin
		finishUC *application.Login

		fixedNow       time.Time
		challengeToken string
		authResp       []byte
		externalID     []byte
		userID         uuid.UUID
		credID         uuid.UUID
	)

	BeforeEach(func() {
		authenticator = mocks.NewAuthenticator(GinkgoT())
		credentials = mocks.NewCredentialFinder(GinkgoT())
		credentialSaver = mocks.NewCredentialSaver(GinkgoT())
		wrappedKeys = mocks.NewCredentialWrappedKeysFinder(GinkgoT())
		sessions = mocks.NewRefreshTokenSaver(GinkgoT())
		refreshTokens = mocks.NewTokenGenerator(GinkgoT())
		accessTokens = mocks.NewTokenIssuer(GinkgoT())
		ids = mocks.NewIDGenerator(GinkgoT())
		transactor = mocks.NewTransactor(GinkgoT())
		revokeSessions = mocks.NewActiveSessionsFinder(GinkgoT())

		fixedNow = time.Date(2026, time.August, 4, 12, 0, 0, 0, time.UTC)
		challengeToken = testChallengeToken
		authResp = []byte("valid-webauthn-response")
		externalID = []byte("credential-external-id")
		userID = uuid.New()
		credID = uuid.New()

		injector = do.New(application.Package)

		do.OverrideValue[application.Authenticator](injector, authenticator)
		do.OverrideValue[application.CredentialFinder](injector, credentials)
		do.OverrideValue[application.CredentialSaver](injector, credentialSaver)
		do.OverrideValue[application.CredentialWrappedKeysFinder](injector, wrappedKeys)
		do.OverrideValue[application.RefreshTokenSaver](injector, sessions)
		do.OverrideValue[application.TokenGenerator](injector, refreshTokens)
		do.OverrideValue[application.TokenIssuer](injector, accessTokens)
		do.OverrideValue[application.IDGenerator](injector, ids)
		do.OverrideValue[application.Transactor](injector, transactor)
		do.OverrideValue[application.ActiveSessionsFinder](injector, revokeSessions)

		do.OverrideValue(injector, logger.NewNOPSlog())
		do.OverrideValue(injector, application.Config{
			Session: application.SessionConfig{
				RefreshTTL: 24 * time.Hour,
				AccessTTL:  15 * time.Minute,
			},
		})

		var err error
		beginUC, err = do.Invoke[*application.BeginLogin](injector)
		Expect(err).ToNot(HaveOccurred())

		finishUC, err = do.Invoke[*application.Login](injector)
		Expect(err).ToNot(HaveOccurred())

		transactor.EXPECT().
			RunInTransaction(mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).
			Maybe()
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
		cred, err := credential.New(credID, externalID, userID, []byte("pub-key"), []string{testTransportUSB}, fixedNow)
		Expect(err).ToNot(HaveOccurred())
		cred.SetSignCount(initialSignCount)
		return cred
	}

	newTestWrappedKeys := func(cID uuid.UUID) []*key.WrappedKey {
		k1 := key.Reconstruct(uuid.New(), userID, cID, key.ScopeMain, []byte("main-dek-encrypted"), "AES-256-GCM-KW", fixedNow)
		k2 := key.Reconstruct(uuid.New(), userID, cID, key.ScopeDecoy, []byte("decoy-dek-encrypted"), "AES-256-GCM-KW", fixedNow)
		return []*key.WrappedKey{k1, k2}
	}

	// --- BeginLogin UseCase ---

	Describe("BeginLogin", func() {
		It("successfully returns challengeToken and requestOptions", func(ctx SpecContext) {
			expectedOpts := []byte("req-opts")
			authenticator.EXPECT().Begin(ctx).Return(testChallengeToken, expectedOpts, nil).Once()

			token, opts, err := beginUC.Execute(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(token).To(Equal(testChallengeToken))
			Expect(opts).To(Equal(expectedOpts))
		})

		It("fails when Authenticator.Begin returns an error", func(ctx SpecContext) {
			authErr := errors.New("failed to generate webauthn challenge")
			authenticator.EXPECT().Begin(ctx).Return("", nil, authErr).Once()

			token, opts, err := beginUC.Execute(ctx)

			Expect(err).To(MatchError(authErr))
			Expect(token).To(BeEmpty())
			Expect(opts).To(BeNil())
		})
	})

	// --- FinishLogin UseCase ---

	Describe("FinishLogin", func() {
		It("successfully finishes login: updates sign count, retrieves wrapped keys and issues tokens", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				newSignCount := uint32(42)
				authResult := newAuthResult(newSignCount)
				authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

				cred := newCredential(10)
				credentials.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()

				credentialSaver.EXPECT().Save(ctx, mock.MatchedBy(func(c *credential.Credential) bool {
					return c.SignCount() == newSignCount && c.UserID() == userID
				})).Return(nil).Once()

				keys := newTestWrappedKeys(cred.ID())
				wrappedKeys.EXPECT().FindByCredentialID(ctx, cred.ID()).Return(keys, nil).Once()

				expectedRefreshCode := testRefreshCode
				refreshTokens.EXPECT().GenerateToken(32).Return(expectedRefreshCode, "refresh-hash", nil).Once()

				sessionID := uuid.New()
				ids.EXPECT().NewID().Return(sessionID).Once()
				sessions.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()

				expectedAccessCode := testAccessCode
				accessTokens.EXPECT().IssueAccessToken(userID, 15*time.Minute).Return(expectedAccessCode, nil).Once()

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
				Expect(out.WrappedKeys[1]).To(Equal(application.WrappedKeyOutput{
					Scope:         key.ScopeDecoy,
					WrappedDEK:    []byte("decoy-dek-encrypted"),
					WrapAlgorithm: "AES-256-GCM-KW",
				}))
			})
		})

		It("fails when Authenticator.Finish returns error", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				authErr := errors.New("invalid signature or challenge expired")
				authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(application.AuthenticationResult{}, authErr).Once()

				out, err := finishUC.Execute(ctx, newInput())

				Expect(err).To(MatchError(authErr))
				Expect(out).To(BeNil())
			})
		})

		It("returns ErrCredentialNotFound when external ID is missing in database", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				authResult := newAuthResult(42)
				authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

				credentials.EXPECT().FindByExternalID(ctx, externalID).Return(nil, domain.ErrNotFound).Once()

				out, err := finishUC.Execute(ctx, newInput())

				Expect(err).To(MatchError(application.ErrCredentialNotFound))
				Expect(out).To(BeNil())
			})
		})

		It("returns database error when credential lookup fails", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				authResult := newAuthResult(42)
				authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

				dbErr := errors.New("db connection failure")
				credentials.EXPECT().FindByExternalID(ctx, externalID).Return(nil, dbErr).Once()

				out, err := finishUC.Execute(ctx, newInput())

				Expect(err).To(MatchError(dbErr))
				Expect(out).To(BeNil())
			})
		})

		It("returns ErrCredentialRevoked when credential is already revoked", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				authResult := newAuthResult(42)
				authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

				cred := newCredential(10)
				err := cred.Revoke(fixedNow)
				Expect(err).ToNot(HaveOccurred())
				credentials.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()

				out, err := finishUC.Execute(ctx, newInput())

				Expect(err).To(MatchError(application.ErrCredentialRevoked))
				Expect(out).To(BeNil())
			})
		})

		It("detects clone via CloneWarning, revokes credential and sessions, and returns ErrCredentialCloneSuspected", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				authResult := newAuthResult(42)
				authResult.CloneWarning = true
				authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

				cred := newCredential(10)
				credentials.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()

				credentialSaver.EXPECT().Save(ctx, mock.MatchedBy(func(c *credential.Credential) bool {
					return c.IsRevoked()
				})).Return(nil).Once()

				revokeSessions.EXPECT().FindActiveByUserIDs(ctx, []user.UserID{userID}, mock.Anything).Return(nil, nil).Once()

				out, err := finishUC.Execute(ctx, newInput())

				Expect(err).To(MatchError(application.ErrCredentialCloneSuspected))
				Expect(out).To(BeNil())
			})
		})

		It("detects clone via SignCount regression, revokes credential and sessions, and returns ErrCredentialCloneSuspected", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				authResult := newAuthResult(5) // Значение меньше текущего (10)
				authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

				cred := newCredential(10)
				credentials.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()

				credentialSaver.EXPECT().Save(ctx, mock.MatchedBy(func(c *credential.Credential) bool {
					return c.IsRevoked()
				})).Return(nil).Once()

				revokeSessions.EXPECT().FindActiveByUserIDs(ctx, []user.UserID{userID}, mock.Anything).Return(nil, nil).Once()

				out, err := finishUC.Execute(ctx, newInput())

				Expect(err).To(MatchError(application.ErrCredentialCloneSuspected))
				Expect(out).To(BeNil())
			})
		})

		It("fails when saving revoked credential during clone mitigation", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				authResult := newAuthResult(42)
				authResult.CloneWarning = true
				authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

				cred := newCredential(10)
				credentials.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()

				saveErr := errors.New("failed to save revoked credential")
				credentialSaver.EXPECT().Save(ctx, mock.Anything).Return(saveErr).Once()

				out, err := finishUC.Execute(ctx, newInput())

				Expect(err).To(MatchError(saveErr))
				Expect(out).To(BeNil())
			})
		})

		It("fails when session revocation fails during clone mitigation", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				authResult := newAuthResult(42)
				authResult.CloneWarning = true
				authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

				cred := newCredential(10)
				credentials.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()

				credentialSaver.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()

				revokeErr := errors.New("failed to query active sessions for revocation")
				revokeSessions.EXPECT().FindActiveByUserIDs(ctx, []user.UserID{userID}, mock.Anything).Return(nil, revokeErr).Once()

				out, err := finishUC.Execute(ctx, newInput())

				Expect(err).To(MatchError(revokeErr))
				Expect(out).To(BeNil())
			})
		})

		Context("Transaction Failures & Rollback Checks", func() {
			It("rolls back when CredentialSaver.Save fails on sign count update", func(ctx SpecContext) {
				synctest.Test(testT, func(_ *testing.T) {
					authResult := newAuthResult(42)
					authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

					cred := newCredential(10)
					credentials.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()

					saveErr := errors.New("failed to update credential sign count")
					credentialSaver.EXPECT().Save(ctx, mock.Anything).Return(saveErr).Once()

					out, err := finishUC.Execute(ctx, newInput())

					Expect(err).To(MatchError(saveErr))
					Expect(out).To(BeNil())
				})
			})

			It("rolls back when wrappedKeys.FindByCredentialID fails", func(ctx SpecContext) {
				synctest.Test(testT, func(_ *testing.T) {
					authResult := newAuthResult(42)
					authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

					cred := newCredential(10)
					credentials.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()
					credentialSaver.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()

					wkErr := errors.New("failed to query wrapped keys")
					wrappedKeys.EXPECT().FindByCredentialID(ctx, cred.ID()).Return(nil, wkErr).Once()

					out, err := finishUC.Execute(ctx, newInput())

					Expect(err).To(MatchError(wkErr))
					Expect(out).To(BeNil())
				})
			})

			It("rolls back when RefreshToken generation fails", func(ctx SpecContext) {
				synctest.Test(testT, func(_ *testing.T) {
					authResult := newAuthResult(42)
					authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

					cred := newCredential(10)
					credentials.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()
					credentialSaver.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()
					wrappedKeys.EXPECT().FindByCredentialID(ctx, cred.ID()).Return(newTestWrappedKeys(cred.ID()), nil).Once()

					tokenErr := errors.New("entropy error on refresh token gen")
					refreshTokens.EXPECT().GenerateToken(32).Return("", "", tokenErr).Once()

					out, err := finishUC.Execute(ctx, newInput())

					Expect(err).To(MatchError(tokenErr))
					Expect(out).To(BeNil())
				})
			})

			It("rolls back when IssueAccessToken fails after refresh token creation", func(ctx SpecContext) {
				synctest.Test(testT, func(_ *testing.T) {
					authResult := newAuthResult(42)
					authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

					cred := newCredential(10)
					credentials.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()
					credentialSaver.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()
					wrappedKeys.EXPECT().FindByCredentialID(ctx, cred.ID()).Return(newTestWrappedKeys(cred.ID()), nil).Once()

					refreshTokens.EXPECT().GenerateToken(32).Return("refresh-code", "hash", nil).Once()
					ids.EXPECT().NewID().Return(uuid.New()).Once()
					sessions.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()

					jwtErr := errors.New("failed to sign access token RSA key error")
					accessTokens.EXPECT().IssueAccessToken(userID, 15*time.Minute).Return("", jwtErr).Once()

					out, err := finishUC.Execute(ctx, newInput())

					Expect(err).To(MatchError(jwtErr))
					Expect(out).To(BeNil())
				})
			})
		})
	})
})
