package application_test

import (
	"context"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/mock"

	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/application/mocks"
	"github.com/whicu/hsa/internal/domain/session"
	"github.com/whicu/hsa/internal/domain/user"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("RevokeCompromisedChain", func() {
	var (
		injector          do.Injector
		descendantsFinder *mocks.UserDescendantsFinder
		revokeSessions    *mocks.ActiveSessionsFinder
		revokeSaver       *mocks.RefreshTokenSaver
		transactor        *mocks.Transactor
		useCase           *application.RevokeCompromisedChain

		compromisedUserID user.UserID
		descendant1ID     user.UserID
		descendant2ID     user.UserID
	)

	BeforeEach(func() {
		descendantsFinder = mocks.NewUserDescendantsFinder(GinkgoT())
		revokeSessions = mocks.NewActiveSessionsFinder(GinkgoT())
		revokeSaver = mocks.NewRefreshTokenSaver(GinkgoT())
		transactor = mocks.NewTransactor(GinkgoT())

		compromisedUserID = uuid.New()
		descendant1ID = uuid.New()
		descendant2ID = uuid.New()

		injector = do.New(application.Package)

		do.OverrideValue[application.UserDescendantsFinder](injector, descendantsFinder)
		do.OverrideValue[application.ActiveSessionsFinder](injector, revokeSessions)
		do.OverrideValue[application.RefreshTokenSaver](injector, revokeSaver)
		do.OverrideValue[application.Transactor](injector, transactor)
		do.OverrideValue[*slog.Logger](injector, logger.NewNOPSlog())

		var err error
		useCase, err = do.Invoke[*application.RevokeCompromisedChain](injector)
		Expect(err).ToNot(HaveOccurred())
	})

	// прозрачный passthrough для транзактора внутреннего RevokeAllUserSessions —
	// сама транзакционность там уже покрыта отдельным набором тестов,
	// здесь важно лишь то, что RevokeCompromisedChain корректно строит
	// список userIDs и передаёт его дальше
	expectRevokeTransactionPassthrough := func(ctx context.Context) {
		transactor.EXPECT().
			RunInTransaction(ctx, mock.Anything).
			RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).
			Once()
	}

	Context("Execute", func() {
		It("should revoke sessions for the compromised user and all descendants", func(ctx SpecContext) {
			descendantsFinder.EXPECT().
				Descendants(ctx, compromisedUserID).
				Return([]user.UserID{descendant1ID, descendant2ID}, nil).
				Once()

			expectedUserIDs := []user.UserID{descendant1ID, descendant2ID, compromisedUserID}

			expectRevokeTransactionPassthrough(ctx)

			revokeSessions.EXPECT().
				FindActiveByUserIDs(ctx, expectedUserIDs, mock.AnythingOfType("time.Time")).
				Return([]*session.RefreshToken{}, nil).
				Once()

			err := useCase.Execute(ctx, compromisedUserID)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should revoke sessions for the compromised user alone when there are no descendants", func(ctx SpecContext) {
			descendantsFinder.EXPECT().
				Descendants(ctx, compromisedUserID).
				Return([]user.UserID{}, nil).
				Once()

			expectedUserIDs := []user.UserID{compromisedUserID}

			expectRevokeTransactionPassthrough(ctx)

			revokeSessions.EXPECT().
				FindActiveByUserIDs(ctx, expectedUserIDs, mock.AnythingOfType("time.Time")).
				Return([]*session.RefreshToken{}, nil).
				Once()

			err := useCase.Execute(ctx, compromisedUserID)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error and skip revocation when finding descendants fails", func(ctx SpecContext) {
			findErr := errors.New("descendants lookup failed")

			descendantsFinder.EXPECT().
				Descendants(ctx, compromisedUserID).
				Return(nil, findErr).
				Once()

			// ни revokeTransactor, ни revokeSessions/revokeSaver не должны
			// вызываться вовсе — отсутствие EXPECT() на них само
			// зафейлит тест при неожиданном вызове

			err := useCase.Execute(ctx, compromisedUserID)

			Expect(err).To(MatchError(findErr))
		})

		It("should propagate error when revoking the user chain fails", func(ctx SpecContext) {
			revokeErr := errors.New("revoke chain failed")

			descendantsFinder.EXPECT().
				Descendants(ctx, compromisedUserID).
				Return([]user.UserID{descendant1ID}, nil).
				Once()

			// ошибка происходит на уровне самой транзакции —
			// RevokeAllUserSessions.Execute целиком возвращает revokeErr
			transactor.EXPECT().
				RunInTransaction(ctx, mock.Anything).
				Return(revokeErr).
				Once()

			err := useCase.Execute(ctx, compromisedUserID)

			Expect(err).To(MatchError(revokeErr))
		})
	})
})
