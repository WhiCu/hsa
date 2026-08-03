package application_test

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/application/mocks"
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/credential"
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
		si       *application.SessionIssuer

		ctx context.Context
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

		ctx = context.Background()

		si = application.NewSessionIssuer(sessions, refreshTokens, accessTokens, ids, 24*time.Hour, 15*time.Minute)

		beginUC = application.NewBeginLogin(authenticator)
		finishUC = application.NewLogin(credentials, credentialSaver, authenticator, si, transactor)

		transactor.On("RunInTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).Return(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).Maybe()
	})

	Describe("BeginLogin", func() {
		It("Basic Case: returns challengeToken + requestOptions", func() {
			expectedToken := "challenge-token"
			expectedOpts := []byte("req-opts")
			authenticator.EXPECT().Begin(ctx).Return(expectedToken, expectedOpts, nil).Once()

			token, opts, err := beginUC.Execute(ctx)

			Expect(err).ToNot(HaveOccurred())
			Expect(token).To(Equal(expectedToken))
			Expect(opts).To(Equal(expectedOpts))
		})
	})

	Describe("FinishLogin", func() {
		It("Basic Case: Valid login response, Updates SignCount, returns tokens", func() {
			challengeToken := "challenge-token"
			authResp := []byte("auth-resp")
			externalID := []byte("ext-id")
			userID := uuid.New()
			newSignCount := uint32(42)

			authResult := application.AuthenticationResult{
				UserID:       userID,
				ExternalID:   externalID,
				NewSignCount: newSignCount,
			}
			authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

			cred, _ := credential.New(uuid.New(), externalID, userID, []byte("pub"), []string{"usb"}, time.Now())
			cred.SetSignCount(10) // old sign count

			credentials.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()

			credentialSaver.EXPECT().Save(ctx, mock.MatchedBy(func(c *credential.Credential) bool {
				return c.SignCount() == newSignCount && c.UserID() == userID
			})).Return(nil).Once()

			expectedRefreshCode := "refresh-code"
			refreshTokens.EXPECT().GenerateToken(32).Return(expectedRefreshCode, "refresh-hash", nil).Once()

			sessionID := uuid.New()
			ids.EXPECT().NewID().Return(sessionID).Once()

			sessions.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()

			expectedAccessCode := "access-code"
			accessTokens.EXPECT().IssueAccessToken(userID, 15*time.Minute).Return(expectedAccessCode, nil).Once()

			in := application.LoginInput{
				ChallengeToken:         challengeToken,
				AuthenticationResponse: authResp,
				DeviceInfo:             "device",
				IPAddress:              "127.0.0.1",
			}

			out, err := finishUC.Execute(ctx, in)

			Expect(err).ToNot(HaveOccurred())
			Expect(out).ToNot(BeNil())
			Expect(out.AccessToken).To(Equal(expectedAccessCode))
			Expect(out.RefreshToken).To(Equal(expectedRefreshCode))
		})

		It("Unknown ExternalID: WebAuthn response valid, but ExternalID not found", func() {
			challengeToken := "challenge-token"
			authResp := []byte("auth-resp")
			externalID := []byte("ext-id")

			authResult := application.AuthenticationResult{
				UserID:       uuid.New(),
				ExternalID:   externalID,
				NewSignCount: uint32(42),
			}
			authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

			credentials.EXPECT().FindByExternalID(ctx, externalID).Return(nil, domain.ErrNotFound).Once()

			in := application.LoginInput{
				ChallengeToken:         challengeToken,
				AuthenticationResponse: authResp,
			}

			out, err := finishUC.Execute(ctx, in)

			Expect(err).To(MatchError(application.ErrCredentialNotFound))
			Expect(out).To(BeNil())
		})

		It("Forged ChallengeToken: Tampered HMAC -> fails at Decode step before transaction", func() {
			expectedErr := errors.New("challenge decode error")
			authenticator.EXPECT().Finish(ctx, "forged-token", []byte("resp")).Return(application.AuthenticationResult{}, expectedErr).Once()

			in := application.LoginInput{
				ChallengeToken:         "forged-token",
				AuthenticationResponse: []byte("resp"),
			}

			out, err := finishUC.Execute(ctx, in)

			Expect(err).To(MatchError(expectedErr))
			Expect(out).To(BeNil())
		})

		It("Mid-Transaction Failure: SessionIssuer.Issue fails -> transaction rolls back", func() {
			challengeToken := "challenge-token"
			authResp := []byte("auth-resp")
			externalID := []byte("ext-id")
			userID := uuid.New()

			authResult := application.AuthenticationResult{
				UserID:       userID,
				ExternalID:   externalID,
				NewSignCount: uint32(42),
			}
			authenticator.EXPECT().Finish(ctx, challengeToken, authResp).Return(authResult, nil).Once()

			cred, _ := credential.New(uuid.New(), externalID, userID, []byte("pub"), []string{"usb"}, time.Now())
			credentials.EXPECT().FindByExternalID(ctx, externalID).Return(cred, nil).Once()

			credentialSaver.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()

			// Session issue fails at generate token
			expectedErr := errors.New("token gen failed")
			refreshTokens.EXPECT().GenerateToken(32).Return("", "", expectedErr).Once()

			in := application.LoginInput{
				ChallengeToken:         challengeToken,
				AuthenticationResponse: authResp,
			}

			out, err := finishUC.Execute(ctx, in)

			Expect(err).To(MatchError(expectedErr))
			Expect(out).To(BeNil())
		})
	})
})
