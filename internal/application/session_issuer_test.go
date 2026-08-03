package application_test

import (
	"errors"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/application/mocks"
	"github.com/whicu/hsa/internal/domain/session"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("SessionIssuer", func() {
	var (
		sessions      *mocks.RefreshTokenSaver
		refreshTokens *mocks.TokenGenerator
		accessTokens  *mocks.TokenIssuer
		ids           *mocks.IDGenerator

		refreshTTL time.Duration
		accessTTL  time.Duration
		si         *application.SessionIssuer

		userID     uuid.UUID
		deviceInfo string
		ipAddress  string
		now        time.Time
	)

	BeforeEach(func() {
		sessions = mocks.NewRefreshTokenSaver(GinkgoT())
		refreshTokens = mocks.NewTokenGenerator(GinkgoT())
		accessTokens = mocks.NewTokenIssuer(GinkgoT())
		ids = mocks.NewIDGenerator(GinkgoT())

		refreshTTL = 24 * time.Hour
		accessTTL = 15 * time.Minute

		si = application.NewSessionIssuer(
			logger.NewNOPSlog(),
			sessions,
			refreshTokens,
			accessTokens,
			ids,
			refreshTTL,
			accessTTL,
		)

		userID = uuid.New()
		deviceInfo = "Mozilla/5.0 (Mobile)"
		ipAddress = "127.0.0.1"
		now = time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
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

		sessions.EXPECT().Save(ctx, mock.MatchedBy(func(s *session.RefreshToken) bool {
			return s.ID() == sessionID && s.IsValid(now)
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
})
