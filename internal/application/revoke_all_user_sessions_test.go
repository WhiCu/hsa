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
	"github.com/whicu/hsa/internal/domain/user"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("RevokeAllUserSessions", func() {
	var (
		sessions   *mocks.ActiveSessionsFinder
		saver      *mocks.RefreshTokenSaver
		transactor *mocks.Transactor
		useCase    *application.RevokeAllUserSessions

		user1ID user.UserID
		user2ID user.UserID
	)

	BeforeEach(func() {
		sessions = mocks.NewActiveSessionsFinder(GinkgoT())
		saver = mocks.NewRefreshTokenSaver(GinkgoT())
		transactor = mocks.NewTransactor(GinkgoT())

		useCase = application.NewRevokeAllUserSessions(logger.NewNOPSlog(), sessions, saver, transactor)

		user1ID = uuid.New()
		user2ID = uuid.New()
	})

	expectTransactionPassthrough := func(ctx context.Context) {
		transactor.EXPECT().
			RunInTransaction(ctx, mock.Anything).
			RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx)
			}).
			Once()
	}

	newToken := func(uid user.UserID) *session.RefreshToken {
		token, err := session.New(
			uuid.New(), uid,
			"test-token-hash", "test-device", "127.0.0.1",
			time.Hour, time.Now(),
		)
		Expect(err).NotTo(HaveOccurred())
		return token
	}

	Context("Execute", func() {
		It("should successfully find, revoke, and save active sessions inside a transaction", func(ctx SpecContext) {
			userIDs := []user.UserID{user1ID, user2ID}
			token := newToken(user1ID)

			expectTransactionPassthrough(ctx)

			sessions.EXPECT().
				FindActiveByUserIDs(ctx, userIDs, mock.AnythingOfType("time.Time")).
				Return([]*session.RefreshToken{token}, nil).
				Once()

			saver.EXPECT().
				Save(ctx, mock.MatchedBy(func(tokens []*session.RefreshToken) bool {
					return len(tokens) == 1 && tokens[0].UserID() == user1ID
				})).
				Return(nil).
				Once()

			err := useCase.Execute(ctx, userIDs...)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should complete successfully when no active sessions exist for users", func(ctx SpecContext) {
			userIDs := []user.UserID{user1ID}

			expectTransactionPassthrough(ctx)

			sessions.EXPECT().
				FindActiveByUserIDs(ctx, userIDs, mock.AnythingOfType("time.Time")).
				Return([]*session.RefreshToken{}, nil).
				Once()

			err := useCase.Execute(ctx, userIDs...)

			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error and abort when finding active sessions fails", func(ctx SpecContext) {
			findErr := errors.New("db query failed")
			userIDs := []user.UserID{user1ID}

			expectTransactionPassthrough(ctx)

			sessions.EXPECT().
				FindActiveByUserIDs(ctx, userIDs, mock.AnythingOfType("time.Time")).
				Return(nil, findErr).
				Once()

			// saver.Save намеренно без EXPECT() — если реализация вдруг
			// начнёт звать Save даже после ошибки поиска, мок сам
			// упадёт с "unexpected call", и это будет честной регрессией

			err := useCase.Execute(ctx, userIDs...)

			Expect(err).To(MatchError(findErr))
		})

		It("should return error when batch saving revoked tokens fails", func(ctx SpecContext) {
			saveErr := errors.New("db save batch failed")
			userIDs := []user.UserID{user1ID}
			token := newToken(user1ID)

			expectTransactionPassthrough(ctx)

			sessions.EXPECT().
				FindActiveByUserIDs(ctx, userIDs, mock.AnythingOfType("time.Time")).
				Return([]*session.RefreshToken{token}, nil).
				Once()

			saver.EXPECT().
				Save(ctx, mock.Anything).
				Return(saveErr).
				Once()

			err := useCase.Execute(ctx, userIDs...)

			Expect(err).To(MatchError(saveErr))
		})

		It("should propagate error directly if transaction execution fails", func(ctx SpecContext) {
			txErr := errors.New("transaction start failed")
			userIDs := []user.UserID{user1ID}

			// здесь transactor не прокидывает fn вовсе (например, ошибка
			// BEGIN) — sessions/saver не должны вызываться совсем, что и
			// проверяется отсутствием EXPECT() на них
			transactor.EXPECT().
				RunInTransaction(ctx, mock.Anything).
				Return(txErr).
				Once()

			err := useCase.Execute(ctx, userIDs...)

			Expect(err).To(MatchError(txErr))
		})
	})
})
