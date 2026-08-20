package application_test

import (
	"context"
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
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/session"
	"github.com/whicu/hsa/internal/domain/user"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("RevokeSession", func() {
	var (
		injector   do.Injector
		finder     *mocks.RefreshTokenFinderByID
		saver      *mocks.RefreshTokenSaver
		transactor *mocks.Transactor
		uc         *application.RevokeSession

		sessionID uuid.UUID
		userID    uuid.UUID
		fixedNow  time.Time
	)

	BeforeEach(func() {
		finder = mocks.NewRefreshTokenFinderByID(GinkgoT())
		saver = mocks.NewRefreshTokenSaver(GinkgoT())
		transactor = mocks.NewTransactor(GinkgoT())

		sessionID = uuid.New()
		userID = uuid.New()
		fixedNow = time.Date(2026, time.August, 20, 10, 0, 0, 0, time.UTC)

		injector = do.New(application.Package)

		do.OverrideValue[application.RefreshTokenFinderByID](injector, finder)
		do.OverrideValue[application.RefreshTokenSaver](injector, saver)
		do.OverrideValue[application.Transactor](injector, transactor)
		do.OverrideValue(injector, logger.NewNOPSlog())

		var err error
		uc, err = do.Invoke[*application.RevokeSession](injector)
		Expect(err).ToNot(HaveOccurred())

		// По умолчанию проксируем выполнение через транзакцию
		transactor.EXPECT().
			RunInTransaction(mock.Anything, mock.AnythingOfType("func(context.Context) error")).
			RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).
			Maybe()
	})

	newTestSession := func(ownerID user.UserID, isRevoked bool) *session.RefreshToken {
		var revokedAt *time.Time
		if isRevoked {
			t := fixedNow.Add(-10 * time.Minute)
			revokedAt = &t
		}

		return session.Reconstruct(
			sessionID,
			ownerID,
			"sha256_hash_value",
			"Chrome / Windows",
			netip.MustParseAddr("127.0.0.1"),
			fixedNow.Add(24*time.Hour),
			revokedAt,
			fixedNow.Add(-time.Hour),
		)
	}

	Describe("Execute", func() {
		It("successfully revokes an active session belonging to the user", func(ctx SpecContext) {
			activeSession := newTestSession(userID, false)

			finder.EXPECT().
				FindByID(mock.Anything, sessionID).
				Return(activeSession, nil).
				Once()

			saver.EXPECT().
				Save(mock.Anything, mock.MatchedBy(func(tokens []*session.RefreshToken) bool {
					return len(tokens) == 1 &&
						tokens[0].ID() == sessionID &&
						tokens[0].RevokedAt() != nil
				})).
				Return(nil).
				Once()

			err := uc.Execute(ctx, sessionID, userID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("handles already revoked session as a no-op without saving", func(ctx SpecContext) {
			alreadyRevokedSession := newTestSession(userID, true)

			finder.EXPECT().
				FindByID(mock.Anything, sessionID).
				Return(alreadyRevokedSession, nil).
				Once()

			// saver.Save не должен вызываться, так как сессия уже была отозвана
			err := uc.Execute(ctx, sessionID, userID)
			Expect(err).NotTo(HaveOccurred())
		})

		It("returns ErrSessionNotFound when session is missing in database", func(ctx SpecContext) {
			finder.EXPECT().
				FindByID(mock.Anything, sessionID).
				Return(nil, domain.ErrNotFound).
				Once()

			err := uc.Execute(ctx, sessionID, userID)
			Expect(err).To(MatchError(application.ErrSessionNotFound))
		})

		It("returns ErrSessionForbidden when attempting to revoke another user's session", func(ctx SpecContext) {
			otherUserID := uuid.New()
			otherUserSession := newTestSession(otherUserID, false)

			finder.EXPECT().
				FindByID(mock.Anything, sessionID).
				Return(otherUserSession, nil).
				Once()

			err := uc.Execute(ctx, sessionID, userID)
			Expect(err).To(MatchError(application.ErrSessionForbidden))
		})

		It("fails when session lookup returns database error", func(ctx SpecContext) {
			dbErr := errors.New("database connection failed")

			finder.EXPECT().
				FindByID(mock.Anything, sessionID).
				Return(nil, dbErr).
				Once()

			err := uc.Execute(ctx, sessionID, userID)
			Expect(err).To(MatchError(dbErr))
		})

		It("fails when saving revoked session returns database error", func(ctx SpecContext) {
			activeSession := newTestSession(userID, false)
			saveErr := errors.New("failed to update session row")

			finder.EXPECT().
				FindByID(mock.Anything, sessionID).
				Return(activeSession, nil).
				Once()

			saver.EXPECT().
				Save(mock.Anything, mock.Anything).
				Return(saveErr).
				Once()

			err := uc.Execute(ctx, sessionID, userID)
			Expect(err).To(MatchError(saveErr))
		})
	})

	Describe("DI Container Resolution", func() {
		It("successfully resolves *RevokeSession from injector", func() {
			resolved, err := do.Invoke[*application.RevokeSession](injector)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).NotTo(BeNil())
		})
	})
})
