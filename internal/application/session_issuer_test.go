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
	"github.com/whicu/hsa/internal/domain/session"
	"github.com/whicu/hsa/internal/domain/user"
)

var _ = Describe("SessionIssuer", func() {
	var (
		injector do.Injector
		m        *Mocks // Используем Mocks из MockPackage
		si       *application.SessionIssuer

		refreshTTL time.Duration
		accessTTL  time.Duration

		userID     uuid.UUID
		deviceInfo string
		ipAddress  netip.Addr
		now        time.Time
		testUser   *user.User
	)

	BeforeEach(func() {
		refreshTTL = 24 * time.Hour
		accessTTL = 15 * time.Minute

		userID = uuid.New()
		deviceInfo = "Mozilla/5.0 (Mobile)"
		ipAddress = netip.MustParseAddr("192.168.1.100")
		now = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

		testUser = user.Reconstruct(userID, user.Member, nil, now)

		injector = do.New(application.Package)
		m = MockPackage(injector, GinkgoT())

		cfg := defaultCfg
		cfg.Session = application.SessionConfig{
			RefreshTTL: refreshTTL,
			AccessTTL:  accessTTL,
		}
		do.OverrideValue(injector, cfg)

		var err error
		si, err = do.Invoke[*application.SessionIssuer](injector)
		Expect(err).ToNot(HaveOccurred())
	})

	// --- Helper Functions ---

	expectRefreshGenOK := func(code, hash string) {
		m.TokenGenerator.EXPECT().GenerateToken(32).Return(code, hash, nil).Once()
	}

	expectUserFoundOK := func() {
		// Используем mock.Anything для контекста, так как внутри транзакции
		// контекст может быть обернут (txCtx)
		m.UserFinderByID.EXPECT().FindByID(mock.Anything, userID).Return(testUser, nil).Once()
	}

	expectUserFoundFail := func(expectedErr error) {
		m.UserFinderByID.EXPECT().FindByID(mock.Anything, userID).Return(nil, expectedErr).Once()
	}

	expectAccessIssueOK := func(code string) {
		expectUserFoundOK()
		m.TokenIssuer.EXPECT().IssueAccessToken(userID, testUser.Role(), accessTTL).Return(code, nil).Once()
	}

	expectAccessIssueFail := func(expectedErr error) {
		expectUserFoundOK()
		m.TokenIssuer.EXPECT().IssueAccessToken(userID, testUser.Role(), accessTTL).Return("", expectedErr).Once()
	}

	// --- Successful Issuance ---

	It("Issues access and refresh tokens successfully", func(ctx SpecContext) {
		sessionID := uuid.New()
		expectedRefreshCode := testRefreshCode
		expectedRefreshHash := testRefreshHash
		expectedAccessCode := testAccessCode

		expectRefreshGenOK(expectedRefreshCode, expectedRefreshHash)
		m.IDGenerator.EXPECT().NewID().Return(sessionID).Once()

		m.RefreshTokenSaver.EXPECT().Save(mock.Anything, mock.MatchedBy(func(s []*session.RefreshToken) bool {
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

			m.TokenGenerator.EXPECT().GenerateToken(32).Return("", "", expectedErr).Once()

			access, refresh, err := si.Issue(ctx, userID, deviceInfo, ipAddress, now)

			Expect(err).To(MatchError(expectedErr))
			Expect(access).To(BeEmpty())
			Expect(refresh).To(BeEmpty())
		})

		It("Returns error when session creation fails due to domain validation (empty token hash)", func(ctx SpecContext) {
			expectRefreshGenOK("refresh-code", "")
			m.IDGenerator.EXPECT().NewID().Return(uuid.New()).Once()

			access, refresh, err := si.Issue(ctx, userID, deviceInfo, ipAddress, now)

			Expect(err).To(MatchError(session.ErrTokenHashRequired))
			Expect(access).To(BeEmpty())
			Expect(refresh).To(BeEmpty())
		})

		It("Returns error and rolls back when refresh token save fails", func(ctx SpecContext) {
			sessionID := uuid.New()
			expectRefreshGenOK("refresh-code", "refresh-hash")
			m.IDGenerator.EXPECT().NewID().Return(sessionID).Once()

			expectedErr := errors.New("session save failed")
			m.RefreshTokenSaver.EXPECT().Save(mock.Anything, mock.Anything).Return(expectedErr).Once()

			access, refresh, err := si.Issue(ctx, userID, deviceInfo, ipAddress, now)

			Expect(err).To(MatchError(expectedErr))
			Expect(access).To(BeEmpty())
			Expect(refresh).To(BeEmpty())
		})

		It("Returns error and rolls back when access token issue fails", func(ctx SpecContext) {
			sessionID := uuid.New()
			expectRefreshGenOK("refresh-code", "refresh-hash")
			m.IDGenerator.EXPECT().NewID().Return(sessionID).Once()
			m.RefreshTokenSaver.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

			expectedErr := errors.New("access token issue failed")
			expectAccessIssueFail(expectedErr)

			access, refresh, err := si.Issue(ctx, userID, deviceInfo, ipAddress, now)

			Expect(err).To(MatchError(expectedErr))
			Expect(access).To(BeEmpty())
			Expect(refresh).To(BeEmpty())
		})

		It("Returns error and rolls back when user lookup fails", func(ctx SpecContext) {
			sessionID := uuid.New()
			expectRefreshGenOK("refresh-code", "refresh-hash")
			m.IDGenerator.EXPECT().NewID().Return(sessionID).Once()
			m.RefreshTokenSaver.EXPECT().Save(mock.Anything, mock.Anything).Return(nil).Once()

			expectedErr := errors.New("user lookup failed")
			expectUserFoundFail(expectedErr)

			access, refresh, err := si.Issue(ctx, userID, deviceInfo, ipAddress, now)

			Expect(err).To(HaveOccurred())
			Expect(access).To(BeEmpty())
			Expect(refresh).To(BeEmpty())
		})
	})

	// --- Rotate Scenarios ---

	Context("Rotate", func() {
		var (
			oldRT        *session.RefreshToken
			oldSessionID uuid.UUID
		)

		BeforeEach(func() {
			oldSessionID = uuid.New()
			var err error
			// Создаем валидный старый токен для тестов ротации
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
			m.TokenGenerator.EXPECT().GenerateToken(32).Return(expectedRefreshCode, expectedRefreshHash, nil).Once()

			// 2. Успешная генерация нового ID
			m.IDGenerator.EXPECT().NewID().Return(newSessionID).Once()

			// 3. Выпуск Access-токена
			expectAccessIssueOK(expectedAccessCode)

			// 4. Сохранение ОБОИХ токенов (старого отозванного и нового активного)
			m.RefreshTokenSaver.EXPECT().Save(mock.Anything, mock.MatchedBy(func(s []*session.RefreshToken) bool {
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

				m.TokenGenerator.EXPECT().GenerateToken(32).Return("", "", expectedErr).Once()

				access, refresh, err := si.Rotate(ctx, oldRT, now)

				Expect(err).To(MatchError(expectedErr))
				Expect(access).To(BeEmpty())
				Expect(refresh).To(BeEmpty())
			})

			It("Returns error when domain rotation fails (e.g. empty token hash)", func(ctx SpecContext) {
				m.TokenGenerator.EXPECT().GenerateToken(32).Return("new-code", "", nil).Once()
				m.IDGenerator.EXPECT().NewID().Return(uuid.New()).Once()

				access, refresh, err := si.Rotate(ctx, oldRT, now)

				Expect(err).To(MatchError(session.ErrTokenHashRequired))
				Expect(access).To(BeEmpty())
				Expect(refresh).To(BeEmpty())
			})

			It("Returns error when access token issue fails", func(ctx SpecContext) {
				expectedErr := errors.New("access token issue failed")
				newSessionID := uuid.New()

				m.TokenGenerator.EXPECT().GenerateToken(32).Return("new-code", "new-hash", nil).Once()
				m.IDGenerator.EXPECT().NewID().Return(newSessionID).Once()

				expectAccessIssueFail(expectedErr)

				access, refresh, err := si.Rotate(ctx, oldRT, now)

				Expect(err).To(MatchError(expectedErr))
				Expect(access).To(BeEmpty())
				Expect(refresh).To(BeEmpty())
			})

			It("Returns error when saving rotated tokens fails", func(ctx SpecContext) {
				expectedErr := errors.New("save rotated tokens failed")
				newSessionID := uuid.New()

				m.TokenGenerator.EXPECT().GenerateToken(32).Return("new-code", "new-hash", nil).Once()
				m.IDGenerator.EXPECT().NewID().Return(newSessionID).Once()
				expectAccessIssueOK("new-access")

				m.RefreshTokenSaver.EXPECT().Save(mock.Anything, mock.Anything).Return(expectedErr).Once()

				access, refresh, err := si.Rotate(ctx, oldRT, now)

				Expect(err).To(MatchError(expectedErr))
				Expect(access).To(BeEmpty())
				Expect(refresh).To(BeEmpty())
			})
		})
	})
})
