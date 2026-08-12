package application_test

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/mock"

	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/application/mocks"
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/session"
	"github.com/whicu/hsa/internal/domain/user"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("RefreshAccessToken", func() {
	var (
		injector      do.Injector
		sessionFinder *mocks.RefreshTokenFinder
		sessionSaver  *mocks.RefreshTokenSaver
		accessTokens  *mocks.TokenIssuer
		refreshTokens *mocks.TokenGenerator
		hasher        *mocks.HashGenerator
		ids           *mocks.IDGenerator

		revokeSessions *mocks.ActiveSessionsFinder
		transactor     *mocks.Transactor

		useCase *application.RefreshAccessToken

		refreshTTL time.Duration
		accessTTL  time.Duration

		rawRefreshToken string
		tokenHash       string
		userID          user.UserID
	)

	BeforeEach(func() {
		sessionFinder = mocks.NewRefreshTokenFinder(GinkgoT())
		sessionSaver = mocks.NewRefreshTokenSaver(GinkgoT())
		accessTokens = mocks.NewTokenIssuer(GinkgoT())
		refreshTokens = mocks.NewTokenGenerator(GinkgoT())
		hasher = mocks.NewHashGenerator(GinkgoT())
		ids = mocks.NewIDGenerator(GinkgoT())

		revokeSessions = mocks.NewActiveSessionsFinder(GinkgoT())
		transactor = mocks.NewTransactor(GinkgoT())

		refreshTTL = time.Hour
		accessTTL = 15 * time.Minute

		rawRefreshToken = "raw-refresh-token"
		tokenHash = "hashed-refresh-token"
		userID = uuid.New()

		injector = do.New(application.Package)

		do.OverrideValue[application.RefreshTokenFinder](injector, sessionFinder)
		do.OverrideValue[application.RefreshTokenSaver](injector, sessionSaver)
		do.OverrideValue[application.TokenIssuer](injector, accessTokens)
		do.OverrideValue[application.TokenGenerator](injector, refreshTokens)
		do.OverrideValue[application.HashGenerator](injector, hasher)
		do.OverrideValue[application.IDGenerator](injector, ids)
		do.OverrideValue[application.ActiveSessionsFinder](injector, revokeSessions)
		do.OverrideValue[application.Transactor](injector, transactor)

		do.OverrideValue(injector, logger.NewNOPSlog())
		do.OverrideValue(injector, application.Config{
			Session: application.SessionConfig{
				RefreshTTL: refreshTTL,
				AccessTTL:  accessTTL,
			},
		})

		var err error
		useCase, err = do.Invoke[*application.RefreshAccessToken](injector)
		Expect(err).ToNot(HaveOccurred())
	})

	// значения deviceInfo/ipAddress/ttl несущественны для проверок этого
	// юзкейса — важны только userID, tokenHash и статус (revoked/expired)
	newOldToken := func(now time.Time, ttl time.Duration) *session.RefreshToken {
		rt, err := session.New(
			uuid.New(), userID, tokenHash,
			"test-device", "127.0.0.1",
			ttl, now,
		)
		Expect(err).NotTo(HaveOccurred())
		return rt
	}

	expectHashOK := func() {
		hasher.EXPECT().GenerateHash(rawRefreshToken).Return(tokenHash, nil).Once()
	}

	Context("Execute", func() {
		It("should return error when hash generation fails", func(ctx SpecContext) {
			hashErr := errors.New("hash generation failed")
			hasher.EXPECT().GenerateHash(rawRefreshToken).Return("", hashErr).Once()

			accessToken, refreshToken, err := useCase.Execute(ctx, rawRefreshToken)

			Expect(err).To(MatchError(hashErr))
			Expect(accessToken).To(BeEmpty())
			Expect(refreshToken).To(BeEmpty())
		})

		It("should return ErrRefreshTokenNotFound when token hash is not found", func(ctx SpecContext) {
			expectHashOK()
			sessionFinder.EXPECT().
				FindByTokenHash(ctx, tokenHash).
				Return(nil, domain.ErrNotFound).
				Once()

			accessToken, refreshToken, err := useCase.Execute(ctx, rawRefreshToken)

			Expect(err).To(MatchError(application.ErrRefreshTokenNotFound))
			Expect(accessToken).To(BeEmpty())
			Expect(refreshToken).To(BeEmpty())
		})

		It("should propagate unexpected errors from finding the token", func(ctx SpecContext) {
			findErr := errors.New("db query failed")
			expectHashOK()
			sessionFinder.EXPECT().
				FindByTokenHash(ctx, tokenHash).
				Return(nil, findErr).
				Once()

			accessToken, refreshToken, err := useCase.Execute(ctx, rawRefreshToken)

			Expect(err).To(MatchError(findErr))
			Expect(accessToken).To(BeEmpty())
			Expect(refreshToken).To(BeEmpty())
		})

		It("should detect reuse, revoke all user sessions, and return ErrRefreshTokenReuseDetected", func(ctx SpecContext) {
			now := time.Now()
			oldRT := newOldToken(now, refreshTTL)
			Expect(oldRT.Revoke(now)).To(Succeed())

			expectHashOK()
			sessionFinder.EXPECT().
				FindByTokenHash(ctx, tokenHash).
				Return(oldRT, nil).
				Once()

			// цепочка ожиданий "настоящего" RevokeAllUserSessions —
			// успешный отзыв всех сессий пользователя
			transactor.EXPECT().
				RunInTransaction(ctx, mock.Anything).
				RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				}).
				Once()
			revokeSessions.EXPECT().
				FindActiveByUserIDs(ctx, []user.UserID{userID}, mock.AnythingOfType("time.Time")).
				Return([]*session.RefreshToken{}, nil).
				Once()

			accessToken, refreshToken, err := useCase.Execute(ctx, rawRefreshToken)

			Expect(err).To(MatchError(application.ErrRefreshTokenReuseDetected))
			Expect(accessToken).To(BeEmpty())
			Expect(refreshToken).To(BeEmpty())
		})

		It("should propagate the error when revoking sessions after reuse detection fails", func(ctx SpecContext) {
			now := time.Now()
			oldRT := newOldToken(now, refreshTTL)
			Expect(oldRT.Revoke(now)).To(Succeed())

			revokeErr := errors.New("revoke sessions failed")

			expectHashOK()
			sessionFinder.EXPECT().
				FindByTokenHash(ctx, tokenHash).
				Return(oldRT, nil).
				Once()

			transactor.EXPECT().
				RunInTransaction(ctx, mock.Anything).
				Return(revokeErr).
				Once()

			accessToken, refreshToken, err := useCase.Execute(ctx, rawRefreshToken)

			Expect(err).To(MatchError(revokeErr))
			Expect(accessToken).To(BeEmpty())
			Expect(refreshToken).To(BeEmpty())
		})

		It("should return ErrRefreshTokenInvalid when the token is expired", func(ctx SpecContext) {
			now := time.Now()
			// ttl в прошлом -> IsValid(now) == false, при этом токен не revoked
			oldRT := newOldToken(now.Add(-2*time.Hour), time.Hour)

			expectHashOK()
			sessionFinder.EXPECT().
				FindByTokenHash(ctx, tokenHash).
				Return(oldRT, nil).
				Once()

			accessToken, refreshToken, err := useCase.Execute(ctx, rawRefreshToken)

			Expect(err).To(MatchError(application.ErrRefreshTokenInvalid))
			Expect(accessToken).To(BeEmpty())
			Expect(refreshToken).To(BeEmpty())
		})

		It("should successfully rotate tokens on the happy path", func(ctx SpecContext) {
			now := time.Now()
			oldRT := newOldToken(now, refreshTTL)

			newRawRefresh := "new-raw-refresh-token"
			newRefreshHash := "new-refresh-token-hash"
			newSessionID := uuid.New()
			issuedAccessToken := "new-access-token"

			expectHashOK()
			sessionFinder.EXPECT().
				FindByTokenHash(ctx, tokenHash).
				Return(oldRT, nil).
				Once()

			sessionSaver.EXPECT().
				Save(ctx, mock.MatchedBy(func(tokens []*session.RefreshToken) bool {
					if len(tokens) != 2 {
						return false
					}
					oldTokenMatched := tokens[0].ID() == oldRT.ID() && tokens[0].IsRevoked()
					newTokenMatched := tokens[1].ID() == newSessionID &&
						tokens[1].UserID() == userID &&
						!tokens[1].IsRevoked()

					return oldTokenMatched && newTokenMatched
				})).
				Return(nil).
				Once()

			refreshTokens.EXPECT().
				GenerateToken(mock.AnythingOfType("int")).
				Return(newRawRefresh, newRefreshHash, nil).
				Once()

			ids.EXPECT().NewID().Return(newSessionID).Once()

			accessTokens.EXPECT().
				IssueAccessToken(userID, accessTTL).
				Return(issuedAccessToken, nil).
				Once()

			accessToken, refreshToken, err := useCase.Execute(ctx, rawRefreshToken)

			Expect(err).NotTo(HaveOccurred())
			Expect(accessToken).To(Equal(issuedAccessToken))
			Expect(refreshToken).To(Equal(newRawRefresh))
		})

		It("should propagate error when saving the rotated tokens fails", func(ctx SpecContext) {
			now := time.Now()
			oldRT := newOldToken(now, refreshTTL)
			saveErr := errors.New("save tokens failed")

			expectHashOK()
			sessionFinder.EXPECT().FindByTokenHash(ctx, tokenHash).Return(oldRT, nil).Once()

			refreshTokens.EXPECT().GenerateToken(mock.AnythingOfType("int")).Return("new-raw", "new-hash", nil).Once()

			ids.EXPECT().NewID().Return(uuid.New()).Once()

			accessTokens.EXPECT().IssueAccessToken(oldRT.UserID(), mock.Anything).Return("access-token", nil).Once()

			sessionSaver.EXPECT().Save(ctx, mock.Anything).Return(saveErr).Once()

			accessToken, refreshToken, err := useCase.Execute(ctx, rawRefreshToken)

			Expect(err).To(MatchError(saveErr))
			Expect(accessToken).To(BeEmpty())
			Expect(refreshToken).To(BeEmpty())
		})

		It("should propagate error when generating the new refresh token fails", func(ctx SpecContext) {
			now := time.Now()
			oldRT := newOldToken(now, refreshTTL)
			genErr := errors.New("generate refresh token failed")

			expectHashOK()
			sessionFinder.EXPECT().
				FindByTokenHash(ctx, tokenHash).
				Return(oldRT, nil).
				Once()
			refreshTokens.EXPECT().
				GenerateToken(mock.AnythingOfType("int")).
				Return("", "", genErr).
				Once()

			accessToken, refreshToken, err := useCase.Execute(ctx, rawRefreshToken)

			Expect(err).To(MatchError(genErr))
			Expect(accessToken).To(BeEmpty())
			Expect(refreshToken).To(BeEmpty())
		})

		It("should propagate error when the new session entity is invalid", func(ctx SpecContext) {
			now := time.Now()
			oldRT := newOldToken(now, refreshTTL)

			expectHashOK()
			sessionFinder.EXPECT().
				FindByTokenHash(ctx, tokenHash).
				Return(oldRT, nil).
				Once()

			refreshTokens.EXPECT().
				GenerateToken(mock.AnythingOfType("int")).
				Return("new-raw", "new-hash", nil).
				Once()

			// uuid.Nil в качестве нового ID -> session.New вернёт
			// ErrIDRequired, обёрнутую в domain.ErrInvalidArgument
			ids.EXPECT().NewID().Return(uuid.Nil).Once()

			accessToken, refreshToken, err := useCase.Execute(ctx, rawRefreshToken)

			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, session.ErrIDRequired)).To(BeTrue())
			Expect(accessToken).To(BeEmpty())
			Expect(refreshToken).To(BeEmpty())
		})

		It("should propagate error when saving the rotated tokens fails", func(ctx SpecContext) {
			now := time.Now()
			oldRT := newOldToken(now, refreshTTL)
			newSessionID := uuid.New()
			saveErr := errors.New("save new token failed")

			expectHashOK()
			sessionFinder.EXPECT().
				FindByTokenHash(ctx, tokenHash).
				Return(oldRT, nil).
				Once()

			refreshTokens.EXPECT().
				GenerateToken(mock.AnythingOfType("int")).
				Return("new-raw", "new-hash", nil).
				Once()

			ids.EXPECT().NewID().Return(newSessionID).Once()

			accessTokens.EXPECT().
				IssueAccessToken(oldRT.UserID(), accessTTL).
				Return("new-access-token", nil).
				Once()

			sessionSaver.EXPECT().
				Save(ctx, mock.Anything).
				Return(saveErr).
				Once()

			accessToken, refreshToken, err := useCase.Execute(ctx, rawRefreshToken)

			Expect(err).To(MatchError(saveErr))
			Expect(accessToken).To(BeEmpty())
			Expect(refreshToken).To(BeEmpty())
		})

		It("should propagate error when issuing the new access token fails", func(ctx SpecContext) {
			now := time.Now()
			oldRT := newOldToken(now, refreshTTL)
			newSessionID := uuid.New()
			issueErr := errors.New("issue access token failed")

			expectHashOK()
			sessionFinder.EXPECT().FindByTokenHash(ctx, tokenHash).Return(oldRT, nil).Once()

			refreshTokens.EXPECT().GenerateToken(mock.AnythingOfType("int")).Return("new-raw", "new-hash", nil).Once()
			ids.EXPECT().NewID().Return(newSessionID).Once()

			accessTokens.EXPECT().
				IssueAccessToken(oldRT.UserID(), accessTTL).
				Return("", issueErr).
				Once()

			accessToken, refreshToken, err := useCase.Execute(ctx, rawRefreshToken)

			Expect(err).To(MatchError(issueErr))
			Expect(accessToken).To(BeEmpty())
			Expect(refreshToken).To(BeEmpty())
		})
	})
})
