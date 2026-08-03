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
	"github.com/whicu/hsa/internal/domain/session"
)

var _ = Describe("SessionIssuer", func() {
	var (
		sessions      *mocks.RefreshTokenSaver
		refreshTokens *mocks.TokenGenerator
		accessTokens  *mocks.TokenIssuer
		ids           *mocks.IDGenerator

		si *application.SessionIssuer

		ctx    context.Context
		userID uuid.UUID
		now    time.Time
	)

	BeforeEach(func() {
		sessions = mocks.NewRefreshTokenSaver(GinkgoT())
		refreshTokens = mocks.NewTokenGenerator(GinkgoT())
		accessTokens = mocks.NewTokenIssuer(GinkgoT())
		ids = mocks.NewIDGenerator(GinkgoT())

		ctx = context.Background()
		userID = uuid.New()
		now = time.Now()

		si = application.NewSessionIssuer(sessions, refreshTokens, accessTokens, ids, 24*time.Hour, 15*time.Minute)
	})

	It("Issues access and refresh tokens successfully", func() {
		expectedRefreshCode := "refresh-code"
		expectedRefreshHash := "refresh-hash"
		refreshTokens.EXPECT().GenerateToken(32).Return(expectedRefreshCode, expectedRefreshHash, nil).Once()

		sessionID := uuid.New()
		ids.EXPECT().NewID().Return(sessionID).Once()

		sessions.EXPECT().Save(ctx, mock.MatchedBy(func(s *session.RefreshToken) bool {
			return s.IsValid(now)
		})).Return(nil).Once()

		expectedAccessCode := "access-code"
		accessTokens.EXPECT().IssueAccessToken(userID, 15*time.Minute).Return(expectedAccessCode, nil).Once()

		access, refresh, err := si.Issue(ctx, userID, "device", "127.0.0.1", now)

		Expect(err).ToNot(HaveOccurred())
		Expect(access).To(Equal(expectedAccessCode))
		Expect(refresh).To(Equal(expectedRefreshCode))
	})

	It("Returns error when refresh token generation fails", func() {
		expectedErr := errors.New("refresh token gen failed")
		refreshTokens.EXPECT().GenerateToken(32).Return("", "", expectedErr).Once()

		access, refresh, err := si.Issue(ctx, userID, "device", "127.0.0.1", now)

		Expect(err).To(MatchError(expectedErr))
		Expect(access).To(BeEmpty())
		Expect(refresh).To(BeEmpty())
	})

	It("Returns error when refresh token save fails", func() {
		expectedRefreshCode := "refresh-code"
		expectedRefreshHash := "refresh-hash"
		refreshTokens.EXPECT().GenerateToken(32).Return(expectedRefreshCode, expectedRefreshHash, nil).Once()

		sessionID := uuid.New()
		ids.EXPECT().NewID().Return(sessionID).Once()

		expectedErr := errors.New("session save failed")
		sessions.EXPECT().Save(ctx, mock.Anything).Return(expectedErr).Once()

		access, refresh, err := si.Issue(ctx, userID, "device", "127.0.0.1", now)

		Expect(err).To(MatchError(expectedErr))
		Expect(access).To(BeEmpty())
		Expect(refresh).To(BeEmpty())
	})

	It("Returns error when access token issue fails", func() {
		expectedRefreshCode := "refresh-code"
		expectedRefreshHash := "refresh-hash"
		refreshTokens.EXPECT().GenerateToken(32).Return(expectedRefreshCode, expectedRefreshHash, nil).Once()

		sessionID := uuid.New()
		ids.EXPECT().NewID().Return(sessionID).Once()

		sessions.EXPECT().Save(ctx, mock.Anything).Return(nil).Once()

		expectedErr := errors.New("access token issue failed")
		accessTokens.EXPECT().IssueAccessToken(userID, 15*time.Minute).Return("", expectedErr).Once()

		access, refresh, err := si.Issue(ctx, userID, "device", "127.0.0.1", now)

		Expect(err).To(MatchError(expectedErr))
		Expect(access).To(BeEmpty())
		Expect(refresh).To(BeEmpty())
	})
})
