package application_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/mock"

	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/application/mocks"
	"github.com/whicu/hsa/internal/domain/invite"
	"github.com/whicu/hsa/internal/domain/user"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("CreateInvite", func() {
	var (
		injector   do.Injector
		invites    *mocks.InviteSaver
		counter    *mocks.ActiveInviteCounter
		tokens     *mocks.TokenGenerator
		ids        *mocks.IDGenerator
		transactor *mocks.Transactor
		uc         *application.CreateInvite

		createdBy uuid.UUID
		ttl       time.Duration
	)

	BeforeEach(func() {
		// Initialize mocks
		{
			invites = mocks.NewInviteSaver(GinkgoT())
			counter = mocks.NewActiveInviteCounter(GinkgoT())
			tokens = mocks.NewTokenGenerator(GinkgoT())
			ids = mocks.NewIDGenerator(GinkgoT())
			transactor = mocks.NewTransactor(GinkgoT())
		}

		// Initialize use case
		{
			ttl = 24 * time.Hour
			createdBy = uuid.New()

			injector = do.New(application.Package)

			do.OverrideValue[application.InviteSaver](injector, invites)
			do.OverrideValue[application.ActiveInviteCounter](injector, counter)
			do.OverrideValue[application.TokenGenerator](injector, tokens)
			do.OverrideValue[application.IDGenerator](injector, ids)
			do.OverrideValue[application.Transactor](injector, transactor)

			do.OverrideValue(injector, logger.NewNOPSlog())
			do.OverrideValue(injector, application.Config{
				Invite: application.InviteConfig{
					MaxActive: 3,
					TTL:       ttl,
				},
			})

			var err error
			uc, err = do.Invoke[*application.CreateInvite](injector)
			Expect(err).ToNot(HaveOccurred())
		}

	})

	// --- Helper Functions ---

	// Пробрасывает вызов внутрь коллбэка транзакции для обычных тестов
	expectTransactionPassthrough := func(ctx context.Context) {
		transactor.EXPECT().
			RunInTransaction(ctx, mock.AnythingOfType("func(context.Context) error")).
			RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
				return fn(ctx) // Выполняем бизнес-логику
			}).Once()
	}

	expectTokenGenOK := func(code, hash string) {
		tokens.EXPECT().GenerateToken(32).Return(code, hash, nil).Once()
	}

	expectSaveOK := func(ctx context.Context, expectedID uuid.UUID) {
		invites.EXPECT().Save(ctx, mock.MatchedBy(func(inv *invite.Invite) bool {
			return inv.ID() == expectedID && inv.CreatedBy() == createdBy
		})).Return(nil).Once()
	}

	// --- Domain Logic & Limit Checks ---

	DescribeTable("Invite count limit validation",
		func(ctx SpecContext, activeCount int, shouldSucceed bool) {
			synctest.Test(testT, func(_ *testing.T) {
				now := time.Now()

				expectTransactionPassthrough(ctx)

				counter.EXPECT().
					CountActiveByUser(ctx, createdBy, now).
					Return(activeCount, nil).
					Once()

				if !shouldSucceed {
					code, expiresAt, err := uc.Execute(ctx, createdBy)

					Expect(err).To(MatchError(invite.ErrTooManyActive))
					Expect(code).To(BeEmpty())
					Expect(expiresAt.IsZero()).To(BeTrue())
					return
				}

				expectedCode := "secret-code"
				expectedHash := "hash"
				expectedID := uuid.New()

				expectTokenGenOK(expectedCode, expectedHash)
				ids.EXPECT().NewID().Return(expectedID).Once()
				expectSaveOK(ctx, expectedID)

				code, expiresAt, err := uc.Execute(ctx, createdBy)

				Expect(err).ToNot(HaveOccurred())
				Expect(code).To(Equal(expectedCode))
				Expect(expiresAt).To(Equal(now.Add(ttl)))
			})
		},
		Entry("Basic Case: 0 active invites", 0, true),
		Entry("Limit Freed: 2 active invites (expired/redeemed freed up space)", 2, true),
		Entry("Limit Reached: Exactly 3 active invites", 3, false),
		Entry("Over Limit: 4 active invites", 4, false),
	)

	// --- Infrastructure Failures ---

	Context("Infrastructure Failures", func() {
		It("Fails directly if the Transactor fails to start/commit", func(ctx SpecContext) {
			txErr := errors.New("failed to begin transaction")

			// Здесь мы не вызываем функцию внутри, симулируя ошибку на старте БД
			transactor.EXPECT().
				RunInTransaction(ctx, mock.AnythingOfType("func(context.Context) error")).
				Return(txErr).
				Once()

			code, expiresAt, err := uc.Execute(ctx, createdBy)

			Expect(err).To(MatchError(txErr))
			Expect(code).To(BeEmpty())
			Expect(expiresAt.IsZero()).To(BeTrue())
		})

		It("Fails when ActiveInviteCounter returns a database error", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				now := time.Now()
				dbErr := errors.New("db failure: failed to count active invites")

				expectTransactionPassthrough(ctx)

				counter.EXPECT().
					CountActiveByUser(ctx, createdBy, now).
					Return(0, dbErr).
					Once()

				code, expiresAt, err := uc.Execute(ctx, createdBy)

				Expect(err).To(MatchError(dbErr))
				Expect(code).To(BeEmpty())
				Expect(expiresAt.IsZero()).To(BeTrue())
			})
		})

		It("Fails when TokenGenerator returns an entropy error", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				now := time.Now()

				expectTransactionPassthrough(ctx)

				counter.EXPECT().
					CountActiveByUser(ctx, createdBy, now).
					Return(0, nil).
					Once()

				tokenErr := errors.New("crypto entropy error")
				tokens.EXPECT().GenerateToken(32).Return("", "", tokenErr).Once()

				code, expiresAt, err := uc.Execute(ctx, createdBy)

				Expect(err).To(MatchError(tokenErr))
				Expect(code).To(BeEmpty())
				Expect(expiresAt.IsZero()).To(BeTrue())
			})
		})

		It("Fails when InviteSaver fails to persist the invite", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				now := time.Now()

				expectTransactionPassthrough(ctx)

				counter.EXPECT().
					CountActiveByUser(ctx, createdBy, now).
					Return(0, nil).
					Once()

				expectTokenGenOK("secret-code", "hash")

				expectedID := uuid.New()
				ids.EXPECT().NewID().Return(expectedID).Once()

				saveErr := errors.New("db insert failure")
				invites.EXPECT().Save(ctx, mock.Anything).Return(saveErr).Once()

				code, expiresAt, err := uc.Execute(ctx, createdBy)

				Expect(err).To(MatchError(saveErr))
				Expect(code).To(BeEmpty())
				Expect(expiresAt.IsZero()).To(BeTrue())
			})
		})
	})

	Context("Concurrency Control", func() {
		It("Should strictly enforce policy limit under highly concurrent requests via Transaction Locking", MustPassRepeatedly(10), func(ctx SpecContext) {
			var (
				// dbLock имитирует поведение pg_advisory_xact_lock
				dbLock sync.Mutex

				activeInvites int
				successCount  int32
				failCount     int32
			)

			const concurrentRequests = 10
			const limit = 3

			// Настраиваем Транзакцию так, чтобы она БЛОКИРОВАЛАСЬ на время выполнения,
			// точно так же, как это сделает pg_advisory_xact_lock в базе данных.
			transactor.EXPECT().
				RunInTransaction(mock.Anything, mock.AnythingOfType("func(context.Context) error")).
				RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					dbLock.Lock()
					defer dbLock.Unlock()
					return fn(ctx)
				}).Maybe()

			counter.EXPECT().
				CountActiveByUser(mock.Anything, createdBy, mock.Anything).
				RunAndReturn(func(_ context.Context, _ user.UserID, _ time.Time) (int, error) {
					// Имитация сетевой задержки к БД
					time.Sleep(10 * time.Millisecond)
					return activeInvites, nil
				}).Maybe()

			tokens.EXPECT().
				GenerateToken(32).
				RunAndReturn(func(int) (string, string, error) {
					return "code", "hash", nil
				}).Maybe()

			ids.EXPECT().
				NewID().
				RunAndReturn(
					uuid.New,
				).Maybe()

			invites.EXPECT().
				Save(mock.Anything, mock.Anything).
				RunAndReturn(func(_ context.Context, _ *invite.Invite) error {
					activeInvites++
					return nil
				}).Maybe()

			var wg sync.WaitGroup
			startGate := make(chan struct{})

			for range concurrentRequests {
				wg.Go(func() {
					<-startGate
					_, _, err := uc.Execute(ctx, createdBy)
					if err == nil {
						atomic.AddInt32(&successCount, 1)
					} else if errors.Is(err, invite.ErrTooManyActive) {
						atomic.AddInt32(&failCount, 1)
					}
				})
			}

			// Даем горутинам время подготовиться
			time.Sleep(5 * time.Millisecond)

			// Запускаем все запросы одновременно!
			close(startGate)
			wg.Wait()

			// Благодаря dbLock внутри мока Transactor'а, этот тест теперь ПРАВИЛЬНО завершится!
			Expect(int(successCount)).To(Equal(limit), "Should only create exactly the allowed number of invites")
			Expect(int(failCount)).To(Equal(concurrentRequests-limit), "The rest of requests should fail with policy error")
			Expect(activeInvites).To(Equal(limit), "Database should only contain the allowed number of active invites")
		})
	})
})
