package application_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/application/mocks"
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/credential"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("Login", func() {
	var (
		authenticator   *mocks.Authenticator
		credentials     *mocks.CredentialFinder
		credentialSaver *mocks.CredentialSaver
		sessions        *mocks.RefreshTokenSaver
		refreshTokens   *mocks.TokenGenerator
		accessTokens    *mocks.TokenIssuer
		ids             *mocks.IDGenerator
		transactor      *mocks.Transactor

		beginUC  *application.BeginLogin
		finishUC *application.Login

		fixedNow       time.Time
		challengeToken string
		authResp       []byte
		externalID     []byte
		userID         uuid.UUID
	)

	BeforeEach(func() {
		authenticator = mocks.NewAuthenticator(GinkgoT())
		credentials = mocks.NewCredentialFinder(GinkgoT())
		credentialSaver = mocks.NewCredentialSaver(GinkgoT())
		sessions = mocks.NewRefreshTokenSaver(GinkgoT())
		refreshTokens = mocks.NewTokenGenerator(GinkgoT())
		accessTokens = mocks.NewTokenIssuer(GinkgoT())
		ids = mocks.NewIDGenerator(GinkgoT())
		transactor = mocks.NewTransactor(GinkgoT())

		fixedNow = time.Date(2026, time.August, 4, 12, 0, 0, 0, time.UTC)

		challengeToken = "valid-challenge-token"
		authResp = []byte("valid-webauthn-response")
		externalID = []byte("credential-external-id")
		userID = uuid.New()

		si := application.NewSessionIssuer(
			logger.NewNOPSlog(),
			sessions,
			refreshTokens,
			accessTokens,
			ids,
			24*time.Hour,
			15*time.Minute,
		)

		beginUC = application.NewBeginLogin(logger.NewNOPSlog(), authenticator)
		finishUC = application.NewLogin(logger.NewNOPSlog(), credentials, credentialSaver, authenticator, si, transactor)

		// Мок транзакции: прозрачно исполняет переданную функцию
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
			IPAddress:              "192.168.1.1",
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
		cred, err := credential.New(uuid.New(), externalID, userID, []byte("pub-key"), []string{testTransportUSB}, fixedNow)
		Expect(err).ToNot(HaveOccurred())
		cred.SetSignCount(initialSignCount)
		return cred
	}

	// --- BeginLogin Usecase (Без synctest — время не используется) ---

	Describe("BeginLogin", func() {
		It("Basic Case: returns challengeToken + requestOptions", func(ctx SpecContext) {
			expectedToken := testChallengeToken
			expectedOpts := []byte("req-opts")
			authenticator.EXPECT().Begin(ctx).Return(expectedToken, expectedOpts, nil).Once()

			token, opts, err := beginUC.Execute(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(token).To(Equal(expectedToken))
			Expect(opts).To(Equal(expectedOpts))
		})

		It("Fails when Authenticator.Begin returns an error", func(ctx SpecContext) {
			authErr := errors.New("failed to generate webauthn challenge")
			authenticator.EXPECT().Begin(ctx).Return("", nil, authErr).Once()

			token, opts, err := beginUC.Execute(ctx)

			Expect(err).To(MatchError(authErr))
			Expect(token).To(BeEmpty())
			Expect(opts).To(BeNil())
		})
	})

	// --- FinishLogin Usecase (С synctest — т.к. Login.Execute вызывает time.Now()) ---

	Describe("FinishLogin", func() {
		It("Basic Case: Valid login response, Updates SignCount, returns tokens", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				newSignCount := uint32(42)
				authResult := newAuthResult(newSignCount)
				authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

				cred := newCredential(10)
				credentials.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()

				credentialSaver.EXPECT().Save(ctx, mock.MatchedBy(func(c *credential.Credential) bool {
					return c.SignCount() == newSignCount && c.UserID() == userID
				})).Return(nil).Once()

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
			})
		})

		It("Validation: Authenticator.Finish fails (e.g. forged challenge, signature mismatch)", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				authErr := errors.New("invalid signature or challenge expired")
				authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(application.AuthenticationResult{}, authErr).Once()

				out, err := finishUC.Execute(ctx, newInput())

				Expect(err).To(MatchError(authErr))
				Expect(out).To(BeNil())
			})
		})

		It("Unknown ExternalID: WebAuthn response valid, but ExternalID not found in DB", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				authResult := newAuthResult(42)
				authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

				credentials.EXPECT().FindByExternalID(ctx, externalID).Return(nil, domain.ErrNotFound).Once()

				out, err := finishUC.Execute(ctx, newInput())

				Expect(err).To(MatchError(application.ErrCredentialNotFound))
				Expect(out).To(BeNil())
			})
		})

		It("Database Failure: Credential lookup returns infrastructure error", func(ctx SpecContext) {
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

		Context("Transaction Failures & Rollback Checks", func() {
			It("Rolls back when CredentialSaver.Save fails", func(ctx SpecContext) {
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

			It("Rolls back when RefreshToken generation fails", func(ctx SpecContext) {
				synctest.Test(testT, func(_ *testing.T) {
					authResult := newAuthResult(42)
					authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

					cred := newCredential(10)
					credentials.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()
					credentialSaver.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()

					tokenErr := errors.New("entropy error on refresh token gen")
					refreshTokens.EXPECT().GenerateToken(32).Return("", "", tokenErr).Once()

					out, err := finishUC.Execute(ctx, newInput())

					Expect(err).To(MatchError(tokenErr))
					Expect(out).To(BeNil())
				})
			})

			It("Rolls back when IssueAccessToken fails after refresh token creation", func(ctx SpecContext) {
				synctest.Test(testT, func(_ *testing.T) {
					authResult := newAuthResult(42)
					authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

					cred := newCredential(10)
					credentials.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()
					credentialSaver.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()

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
