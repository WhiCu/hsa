package storage_test

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"

	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/user"
	"github.com/whicu/hsa/internal/infrastructure/storage"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("Storage Transactions & RunInTransaction", func() {
	var (
		injector do.Injector
		srg      *storage.Storage
		userRepo *storage.UserRepository
	)

	BeforeEach(func(ctx SpecContext) {
		injector = do.New(storage.Package)

		do.OverrideValue[context.Context](injector, ctx)
		do.OverrideValue(injector, logger.NewNOPSlog())
		do.OverrideValue(injector, globalConfig)

		var err error
		srg, err = do.Invoke[*storage.Storage](injector)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func(_ SpecContext) {
			rep := injector.Shutdown()
			Expect(rep.Succeed).To(BeTrue())
		})

		userRepo, err = do.Invoke[*storage.UserRepository](injector)
		Expect(err).ToNot(HaveOccurred())

		Expect(srg.Up(ctx)).To(Succeed())
		Expect(srg.Reset(ctx)).To(Succeed())
	})

	Describe("RunInTransaction", func() {
		It("commits transaction and persists data when fn returns nil", func(ctx SpecContext) {
			userID := uuid.New()
			now := time.Now().Truncate(time.Microsecond)
			u, err := user.NewRoot(userID, now)
			Expect(err).ToNot(HaveOccurred())

			err = srg.RunInTransaction(ctx, func(txCtx context.Context) error {
				return userRepo.Save(txCtx, u)
			})
			Expect(err).ToNot(HaveOccurred())

			// Проверяем, что пользователь зафиксирован в БД
			descendants, err := userRepo.Descendants(ctx, userID)
			Expect(err).ToNot(HaveOccurred())
			Expect(descendants).To(Equal([]uuid.UUID{userID}))
		})

		It("rolls back transaction when fn returns an error", func(ctx SpecContext) {
			userID := uuid.New()
			customErr := errors.New("business logic failure")
			u, err := user.NewRoot(userID, time.Now())
			Expect(err).ToNot(HaveOccurred())

			err = srg.RunInTransaction(ctx, func(txCtx context.Context) error {
				saveErr := userRepo.Save(txCtx, u)
				Expect(saveErr).ToNot(HaveOccurred())

				return customErr
			})

			// Ошибка должна вернуться вызывающему коду
			Expect(err).To(MatchError(customErr))

			// Запись должна откатиться
			_, err = userRepo.Descendants(ctx, userID)
			Expect(err).To(MatchError(domain.ErrNotFound))
		})

		It("recovers from panic, rolls back transaction and returns panic error", func(ctx SpecContext) {
			userID := uuid.New()
			u, err := user.NewRoot(userID, time.Now())
			Expect(err).ToNot(HaveOccurred())

			err = srg.RunInTransaction(ctx, func(txCtx context.Context) error {
				saveErr := userRepo.Save(txCtx, u)
				Expect(saveErr).ToNot(HaveOccurred())

				panic("unexpected runtime panic")
			})

			// Паника перехвачена и обернута в ошибку
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unexpected runtime panic"))

			// Проверяем откат транзакции
			_, err = userRepo.Descendants(ctx, userID)
			Expect(err).To(MatchError(domain.ErrNotFound))
		})

		It("successfully handles nested / reentrant RunInTransaction calls using the same transaction", func(ctx SpecContext) {
			now := time.Now()
			rootID := uuid.New()
			childID := uuid.New()

			rootUser, err := user.NewRoot(rootID, now)
			Expect(err).ToNot(HaveOccurred())

			childUser, err := user.New(childID, rootID, now)
			Expect(err).ToNot(HaveOccurred())

			err = srg.RunInTransaction(ctx, func(outerCtx context.Context) error {
				Expect(userRepo.Save(outerCtx, rootUser)).To(Succeed())

				// Вложенный вызов RunInTransaction
				return srg.RunInTransaction(outerCtx, func(innerCtx context.Context) error {
					return userRepo.Save(innerCtx, childUser)
				})
			})

			Expect(err).ToNot(HaveOccurred())

			// Обе записи успешно зафиксированы
			descendants, err := userRepo.Descendants(ctx, rootID)
			Expect(err).ToNot(HaveOccurred())
			Expect(descendants).To(ConsistOf(rootID, childID))
		})

		It("rolls back entire nested transaction if inner operation fails", func(ctx SpecContext) {
			now := time.Now()
			rootID := uuid.New()
			childID := uuid.New()
			innerErr := errors.New("inner failure")

			rootUser, _ := user.NewRoot(rootID, now)
			childUser, _ := user.New(childID, rootID, now)

			err := srg.RunInTransaction(ctx, func(outerCtx context.Context) error {
				Expect(userRepo.Save(outerCtx, rootUser)).To(Succeed())

				return srg.RunInTransaction(outerCtx, func(innerCtx context.Context) error {
					Expect(userRepo.Save(innerCtx, childUser)).To(Succeed())
					return innerErr
				})
			})

			Expect(err).To(MatchError(innerErr))

			// Ни root, ни child не должны сохраниться
			_, err = userRepo.Descendants(ctx, rootID)
			Expect(err).To(MatchError(domain.ErrNotFound))

			_, err = userRepo.Descendants(ctx, childID)
			Expect(err).To(MatchError(domain.ErrNotFound))
		})

		It("rolls back cleanly even if context is cancelled during execution", func(ctx SpecContext) {
			cancelCtx, cancel := context.WithCancel(ctx)
			userID := uuid.New()
			u, err := user.NewRoot(userID, time.Now())
			Expect(err).ToNot(HaveOccurred())

			err = srg.RunInTransaction(cancelCtx, func(txCtx context.Context) error {
				Expect(userRepo.Save(txCtx, u)).To(Succeed())

				// Отменяем контекст прямо во время выполнения транзакции
				cancel()
				return txCtx.Err()
			})

			Expect(err).To(MatchError(context.Canceled))

			// Запись должна откатиться
			_, err = userRepo.Descendants(ctx, userID)
			Expect(err).To(MatchError(domain.ErrNotFound))
		})
	})

	Describe("Transaction Isolation & Propagation", func() {
		It("isolates uncommitted changes from base context until transaction is committed", func(ctx SpecContext) {
			userID := uuid.New()
			u, err := user.NewRoot(userID, time.Now())
			Expect(err).ToNot(HaveOccurred())

			err = srg.RunInTransaction(ctx, func(txCtx context.Context) error {
				// Сохраняем пользователя внутри транзакции
				Expect(userRepo.Save(txCtx, u)).To(Succeed())

				// 1. Внутри транзакционного контекста пользователь уже виден
				txDescendants, txErr := userRepo.Descendants(txCtx, userID)
				Expect(txErr).ToNot(HaveOccurred())
				Expect(txDescendants).To(Equal([]uuid.UUID{userID}))

				// 2. В базовом контексте (вне транзакции) пользователь еще НЕ виден
				_, baseErr := userRepo.Descendants(ctx, userID)
				Expect(baseErr).To(MatchError(domain.ErrNotFound))

				return nil
			})
			Expect(err).ToNot(HaveOccurred())

			// 3. После коммита пользователь становится виден в базовом контексте
			descendants, err := userRepo.Descendants(ctx, userID)
			Expect(err).ToNot(HaveOccurred())
			Expect(descendants).To(Equal([]uuid.UUID{userID}))
		})
	})
})
