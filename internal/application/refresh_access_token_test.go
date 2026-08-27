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
	"github.com/whicu/hsa/internal/domain/session"
	"github.com/whicu/hsa/internal/domain/user"
)

var _ = Describe("RefreshAccessToken", func() {
	var (
		injector do.Injector
		m        *Mocks // Доступ ко всем мокам из MockPackage
		useCase  *application.RefreshAccessToken

		refreshTTL time.Duration
		accessTTL  time.Duration

		rawRefreshToken string
		tokenHash       string
		userID          user.UserID
	)

	BeforeEach(func() {
		refreshTTL = time.Hour
		accessTTL = 15 * time.Minute

		rawRefreshToken = "raw-refresh-token"
		tokenHash = "hashed-refresh-token"
		userID = uuid.New()

		injector = do.New(application.Package)
		m = MockPackage(injector, GinkgoT())

		// Переопределяем конфиг для SessionIssuer (который инжектится внутрь юзкейса)
		cfg := defaultCfg
		cfg.Session = application.SessionConfig{
			RefreshTTL: refreshTTL,
			AccessTTL:  accessTTL,
		}
		do.OverrideValue(injector, cfg)

		var err error
		useCase, err = do.Invoke[*application.RefreshAccessToken](injector)
		Expect(err).ToNot(HaveOccurred())
	})

	// Helper: создает валидную доменную сущность RefreshToken
	newOldToken := func(now time.Time, ttl time.Duration) *session.RefreshToken {
		rt, err := session.New(
			uuid.New(), userID, tokenHash,
			"test-device", netip.MustParseAddr("192.168.1.100"),
			ttl, now,
		)
		Expect(err).NotTo(HaveOccurred())
		return rt
	}

	expectHashOK := func() {
		m.HashGenerator.EXPECT().GenerateHash(rawRefreshToken).Return(tokenHash, nil).Once()
	}

	Context("Execute", func() {

		It("should successfully rotate tokens on the happy path", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				now := time.Now()
				oldRT := newOldToken(now, refreshTTL)

				newRawRefresh := "new-raw-refresh-token"
				newRefreshHash := "new-refresh-token-hash"
				newSessionID := uuid.New()
				issuedAccessToken := "new-access-token"
				testUser := user.Reconstruct(userID, user.Member, nil, now)

				expectHashOK()
				m.RefreshTokenFinder.EXPECT().FindByTokenHash(ctx, tokenHash).Return(oldRT, nil).Once()

				// Ожидания для SessionIssuer.Rotate (внутри транзакции)
				m.TokenGenerator.EXPECT().GenerateToken(32).Return(newRawRefresh, newRefreshHash, nil).Once()
				m.IDGenerator.EXPECT().NewID().Return(newSessionID).Once()

				// Выдача AccessToken
				m.UserFinderByID.EXPECT().FindByID(ctx, userID).Return(testUser, nil).Once()
				m.TokenIssuer.EXPECT().IssueAccessToken(userID, user.Member, accessTTL).Return(issuedAccessToken, nil).Once()

				// Сохранение токенов
				m.RefreshTokenSaver.EXPECT().Save(ctx, mock.MatchedBy(func(tokens []*session.RefreshToken) bool {
					if len(tokens) != 2 {
						return false
					}
					oldTokenMatched := tokens[0].ID() == oldRT.ID() && tokens[0].IsRevoked()
					newTokenMatched := tokens[1].ID() == newSessionID &&
						tokens[1].UserID() == userID &&
						!tokens[1].IsRevoked()

					return oldTokenMatched && newTokenMatched
				})).Return(nil).Once()

				accessToken, refreshToken, err := useCase.Execute(ctx, rawRefreshToken)

				Expect(err).NotTo(HaveOccurred())
				Expect(accessToken).To(Equal(issuedAccessToken))
				Expect(refreshToken).To(Equal(newRawRefresh))
			})
		})

		It("should return error when hash generation fails", func(ctx SpecContext) {
			hashErr := errors.New("hash generation failed")
			m.HashGenerator.EXPECT().GenerateHash(rawRefreshToken).Return("", hashErr).Once()

			accessToken, refreshToken, err := useCase.Execute(ctx, rawRefreshToken)

			Expect(err).To(MatchError(hashErr))
			Expect(accessToken).To(BeEmpty())
			Expect(refreshToken).To(BeEmpty())
		})

		It("should return ErrRefreshTokenNotFound when token hash is not found", func(ctx SpecContext) {
			expectHashOK()
			m.RefreshTokenFinder.EXPECT().FindByTokenHash(ctx, tokenHash).Return(nil, domain.ErrNotFound).Once()

			accessToken, refreshToken, err := useCase.Execute(ctx, rawRefreshToken)

			Expect(err).To(MatchError(application.ErrRefreshTokenNotFound))
			Expect(accessToken).To(BeEmpty())
			Expect(refreshToken).To(BeEmpty())
		})

		It("should propagate unexpected errors from finding the token", func(ctx SpecContext) {
			findErr := errors.New("db query failed")
			expectHashOK()
			m.RefreshTokenFinder.EXPECT().FindByTokenHash(ctx, tokenHash).Return(nil, findErr).Once()

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
			m.RefreshTokenFinder.EXPECT().FindByTokenHash(ctx, tokenHash).Return(oldRT, nil).Once()

			// Ожидания для RevokeAllUserSessions.Execute
			// Поскольку транзакция там вложена (или он имеет свой транзактор),
			// мы просто мокаем успешный поиск и сохранение в рамках отзыва.
			m.ActiveSessionsFinder.EXPECT().FindActiveByUserIDs(ctx, []user.UserID{userID}, mock.AnythingOfType("time.Time")).Return([]*session.RefreshToken{}, nil).Once()

			accessToken, refreshToken, err := useCase.Execute(ctx, rawRefreshToken)

			Expect(err).To(MatchError(application.ErrRefreshTokenReuseDetected))
			Expect(accessToken).To(BeEmpty())
			Expect(refreshToken).To(BeEmpty())
		})

		It("should propagate the error when revoking sessions after reuse detection fails", func(ctx SpecContext) {
			now := time.Now()
			oldRT := newOldToken(now, refreshTTL)
			Expect(oldRT.Revoke(now)).To(Succeed())

			revokeErr := errors.New("revoke sessions query failed")

			expectHashOK()
			m.RefreshTokenFinder.EXPECT().FindByTokenHash(ctx, tokenHash).Return(oldRT, nil).Once()

			// Падение внутри RevokeAllUserSessions
			m.ActiveSessionsFinder.EXPECT().FindActiveByUserIDs(ctx, []user.UserID{userID}, mock.AnythingOfType("time.Time")).Return(nil, revokeErr).Once()

			accessToken, refreshToken, err := useCase.Execute(ctx, rawRefreshToken)

			Expect(err).To(MatchError(revokeErr))
			Expect(accessToken).To(BeEmpty())
			Expect(refreshToken).To(BeEmpty())
		})

		It("should return ErrRefreshTokenInvalid when the token is expired", func(ctx SpecContext) {
			now := time.Now()
			// ttl в прошлом -> IsValid(now) == false
			oldRT := newOldToken(now.Add(-2*time.Hour), time.Hour)

			expectHashOK()
			m.RefreshTokenFinder.EXPECT().FindByTokenHash(ctx, tokenHash).Return(oldRT, nil).Once()

			accessToken, refreshToken, err := useCase.Execute(ctx, rawRefreshToken)

			Expect(err).To(MatchError(application.ErrRefreshTokenInvalid))
			Expect(accessToken).To(BeEmpty())
			Expect(refreshToken).To(BeEmpty())
		})

		It("should roll back transaction and propagate error when SessionIssuer.Rotate fails", func(ctx SpecContext) {
			now := time.Now()
			oldRT := newOldToken(now, refreshTTL)
			genErr := errors.New("token generator failed")

			expectHashOK()
			m.RefreshTokenFinder.EXPECT().FindByTokenHash(ctx, tokenHash).Return(oldRT, nil).Once()

			// Ломаем генерацию токена внутри вложенного вызова SessionIssuer.Rotate
			m.TokenGenerator.EXPECT().GenerateToken(32).Return("", "", genErr).Once()

			accessToken, refreshToken, err := useCase.Execute(ctx, rawRefreshToken)

			Expect(err).To(MatchError(genErr))
			Expect(accessToken).To(BeEmpty())
			Expect(refreshToken).To(BeEmpty())
		})
	})
})
