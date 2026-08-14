package application_test

import (
	"errors"
	"net/netip"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/mock"

	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/application/mocks"
	"github.com/whicu/hsa/internal/domain/session"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("SessionIssuer", func() {
	var (
		injector      do.Injector
		sessions      *mocks.RefreshTokenSaver
		refreshTokens *mocks.TokenGenerator
		accessTokens  *mocks.TokenIssuer
		ids           *mocks.IDGenerator

		refreshTTL time.Duration
		accessTTL  time.Duration
		si         *application.SessionIssuer

		userID     uuid.UUID
		deviceInfo string
		ipAddress  netip.Addr
		now        time.Time
	)

	BeforeEach(func() {
		sessions = mocks.NewRefreshTokenSaver(GinkgoT())
		refreshTokens = mocks.NewTokenGenerator(GinkgoT())
		accessTokens = mocks.NewTokenIssuer(GinkgoT())
		ids = mocks.NewIDGenerator(GinkgoT())

		refreshTTL = 24 * time.Hour
		accessTTL = 15 * time.Minute

		userID = uuid.New()
		deviceInfo = "Mozilla/5.0 (Mobile)"
		ipAddress = netip.MustParseAddr("192.168.1.100")
		now = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

		injector = do.New(application.Package)

		do.OverrideValue[application.RefreshTokenSaver](injector, sessions)
		do.OverrideValue[application.TokenGenerator](injector, refreshTokens)
		do.OverrideValue[application.TokenIssuer](injector, accessTokens)
		do.OverrideValue[application.IDGenerator](injector, ids)

		do.OverrideValue(injector, logger.NewNOPSlog())
		do.OverrideValue(injector, application.Config{
			Session: application.SessionConfig{
				RefreshTTL: refreshTTL,
				AccessTTL:  accessTTL,
			},
		})

		var err error
		si, err = do.Invoke[*application.SessionIssuer](injector)
		Expect(err).ToNot(HaveOccurred())
	})
	// --- Helper Functions ---

	expectRefreshGenOK := func(code, hash string) {
		refreshTokens.EXPECT().GenerateToken(32).Return(code, hash, nil).Once()
	}

	expectAccessIssueOK := func(code string) {
		accessTokens.EXPECT().IssueAccessToken(userID, accessTTL).Return(code, nil).Once()
	}

	// --- Successful Issuance ---

	It("Issues access and refresh tokens successfully", func(ctx SpecContext) {
		sessionID := uuid.New()
		expectedRefreshCode := testRefreshCode
		expectedRefreshHash := testRefreshHash
		expectedAccessCode := testAccessCode

		expectRefreshGenOK(expectedRefreshCode, expectedRefreshHash)
		ids.EXPECT().NewID().Return(sessionID).Once()

		sessions.EXPECT().Save(ctx, mock.MatchedBy(func(s []*session.RefreshToken) bool {
			if len(s) != 1 {
				return false
			}
			return s[0].ID() == sessionID && s[0].IsValid(now)
		})).Return(nil).Once()

		expectAccessIssueOK(expectedAccessCode)

		access, refresh, err := si.Issue(ctx, userID, deviceInfo, ipAddress, now)

		Expect(err).ToNot(HaveOccurred())
		Expect(access).To(Equal(expectedAccessCode))
		Expect(refresh).To(Equal(expectedRefreshCode))
	})

	// --- Failure Scenarios ---

	Context("Failure Scenarios", func() {
		It("Returns error when refresh token generation fails", func(ctx SpecContext) {
			expectedErr := errors.New("refresh token gen failed")

			refreshTokens.EXPECT().GenerateToken(32).Return("", "", expectedErr).Once()

			access, refresh, err := si.Issue(ctx, userID, deviceInfo, ipAddress, now)

			Expect(err).To(MatchError(expectedErr))
			Expect(access).To(BeEmpty())
			Expect(refresh).To(BeEmpty())
		})

		It("Returns error when session creation fails due to domain validation (empty token hash)", func(ctx SpecContext) {
			expectRefreshGenOK("refresh-code", "")
			ids.EXPECT().NewID().Return(uuid.New()).Once()

			access, refresh, err := si.Issue(ctx, userID, deviceInfo, ipAddress, now)

			Expect(err).To(MatchError(session.ErrTokenHashRequired))
			Expect(access).To(BeEmpty())
			Expect(refresh).To(BeEmpty())
		})

		It("Returns error when refresh token save fails", func(ctx SpecContext) {
			sessionID := uuid.New()
			expectRefreshGenOK("refresh-code", "refresh-hash")
			ids.EXPECT().NewID().Return(sessionID).Once()

			expectedErr := errors.New("session save failed")
			sessions.EXPECT().Save(ctx, mock.Anything).Return(expectedErr).Once()

			access, refresh, err := si.Issue(ctx, userID, deviceInfo, ipAddress, now)

			Expect(err).To(MatchError(expectedErr))
			Expect(access).To(BeEmpty())
			Expect(refresh).To(BeEmpty())
		})

		It("Returns error when access token issue fails", func(ctx SpecContext) {
			sessionID := uuid.New()
			expectRefreshGenOK("refresh-code", "refresh-hash")
			ids.EXPECT().NewID().Return(sessionID).Once()
			sessions.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()

			expectedErr := errors.New("access token issue failed")
			accessTokens.EXPECT().IssueAccessToken(userID, accessTTL).Return("", expectedErr).Once()

			access, refresh, err := si.Issue(ctx, userID, deviceInfo, ipAddress, now)

			Expect(err).To(MatchError(expectedErr))
			Expect(access).To(BeEmpty())
			Expect(refresh).To(BeEmpty())
		})
	})
	Context("Rotate", func() {
		var (
			oldRT        *session.RefreshToken
			oldSessionID uuid.UUID
		)

		BeforeEach(func() {
			oldSessionID = uuid.New()
			var err error
			// Создаем валидный старый токен для тестов ротации
			// Время создания делаем в прошлом (на 1 час назад от now)
			oldRT, err = session.New(
				oldSessionID,
				userID,
				"old-hash",
				deviceInfo,
				ipAddress,
				refreshTTL,
				now.Add(-time.Hour),
			)
			Expect(err).ToNot(HaveOccurred())
		})

		It("Rotates access and refresh tokens successfully", func(ctx SpecContext) {
			newSessionID := uuid.New()
			expectedRefreshCode := "new-refresh-code"
			expectedRefreshHash := "new-refresh-hash"
			expectedAccessCode := "new-access-code"

			// 1. Успешная генерация токена
			refreshTokens.EXPECT().GenerateToken(32).Return(expectedRefreshCode, expectedRefreshHash, nil).Once()

			// 2. Успешная генерация нового ID
			ids.EXPECT().NewID().Return(newSessionID).Once()

			// 3. Выпуск Access-токена
			accessTokens.EXPECT().IssueAccessToken(userID, accessTTL).Return(expectedAccessCode, nil).Once()

			// 4. Сохранение ОБОИХ токенов (старого отозванного и нового активного)
			sessions.EXPECT().Save(ctx, mock.MatchedBy(func(s []*session.RefreshToken) bool {
				if len(s) != 2 {
					return false
				}

				oldTokenValid := s[0].ID() == oldSessionID && s[0].IsRevoked()
				newTokenValid := s[1].ID() == newSessionID && !s[1].IsRevoked() && s[1].UserID() == userID

				return oldTokenValid && newTokenValid
			})).Return(nil).Once()

			access, refresh, err := si.Rotate(ctx, oldRT, now)

			Expect(err).ToNot(HaveOccurred())
			Expect(access).To(Equal(expectedAccessCode))
			Expect(refresh).To(Equal(expectedRefreshCode))
		})

		Context("Failure Scenarios", func() {
			It("Returns error when refresh token generation fails", func(ctx SpecContext) {
				expectedErr := errors.New("refresh token gen failed")

				// Падаем на первом же шаге
				refreshTokens.EXPECT().GenerateToken(32).Return("", "", expectedErr).Once()

				access, refresh, err := si.Rotate(ctx, oldRT, now)

				Expect(err).To(MatchError(expectedErr))
				Expect(access).To(BeEmpty())
				Expect(refresh).To(BeEmpty())
			})

			It("Returns error when domain rotation fails (e.g. empty token hash)", func(ctx SpecContext) {
				// Возвращаем пустой хэш, чтобы доменная сущность вернула ошибку при создании
				refreshTokens.EXPECT().GenerateToken(32).Return("new-code", "", nil).Once()
				ids.EXPECT().NewID().Return(uuid.New()).Once()

				access, refresh, err := si.Rotate(ctx, oldRT, now)

				Expect(err).To(MatchError(session.ErrTokenHashRequired))
				Expect(access).To(BeEmpty())
				Expect(refresh).To(BeEmpty())
			})

			It("Returns error when access token issue fails", func(ctx SpecContext) {
				expectedErr := errors.New("access token issue failed")
				newSessionID := uuid.New()

				// Проходим генерацию
				refreshTokens.EXPECT().GenerateToken(32).Return("new-code", "new-hash", nil).Once()
				ids.EXPECT().NewID().Return(newSessionID).Once()

				// Падаем на выпуске access-токена
				accessTokens.EXPECT().IssueAccessToken(userID, accessTTL).Return("", expectedErr).Once()

				access, refresh, err := si.Rotate(ctx, oldRT, now)

				Expect(err).To(MatchError(expectedErr))
				Expect(access).To(BeEmpty())
				Expect(refresh).To(BeEmpty())
			})

			It("Returns error when saving rotated tokens fails", func(ctx SpecContext) {
				expectedErr := errors.New("save rotated tokens failed")
				newSessionID := uuid.New()

				// Проходим генерацию
				refreshTokens.EXPECT().GenerateToken(32).Return("new-code", "new-hash", nil).Once()
				ids.EXPECT().NewID().Return(newSessionID).Once()

				// Проходим выпуск access-токена
				accessTokens.EXPECT().IssueAccessToken(userID, accessTTL).Return("new-access", nil).Once()

				// Падаем на сохранении базы
				sessions.EXPECT().Save(ctx, mock.Anything).Return(expectedErr).Once()

				access, refresh, err := si.Rotate(ctx, oldRT, now)

				Expect(err).To(MatchError(expectedErr))
				Expect(access).To(BeEmpty())
				Expect(refresh).To(BeEmpty())
			})
		})
	})
})
