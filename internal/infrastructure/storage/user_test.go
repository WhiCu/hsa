package storage_test

import (
	"context"
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

var _ = Describe("UserRepository", func() {
	var (
		injector do.Injector
		st       *storage.Storage
		userRepo *storage.UserRepository
	)

	BeforeEach(func(ctx SpecContext) {
		// 1. Инициализируем чистый DI-контейнер
		injector = do.New(storage.Package)

		do.OverrideValue(injector, logger.NewNOPSlog())
		do.OverrideValue(injector, globalConfig)
		do.OverrideValue[context.Context](injector, ctx)

		// 2. Резолвим зависимости
		var err error
		st, err = do.Invoke[*storage.Storage](injector)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func(_ SpecContext) {
			rep := injector.Shutdown()
			Expect(rep.Succeed).To(BeTrue())
		})

		userRepo, err = do.Invoke[*storage.UserRepository](injector)
		Expect(err).ToNot(HaveOccurred())

		// 3. Выполняем миграции и очищаем таблицы перед каждым тестом
		Expect(st.Up(ctx)).To(Succeed())
		Expect(st.Reset(ctx)).To(Succeed())
	})

	Describe("Save", func() {
		It("successfully saves a root user", func(ctx SpecContext) {
			u, err := user.NewRoot(uuid.New(), time.Now())
			Expect(err).ToNot(HaveOccurred())

			Expect(userRepo.Save(ctx, u)).To(Succeed())
		})

		It("successfully saves a child user referencing parent", func(ctx SpecContext) {
			parentID := uuid.New()
			parent, err := user.NewRoot(parentID, time.Now())
			Expect(err).ToNot(HaveOccurred())
			Expect(userRepo.Save(ctx, parent)).To(Succeed())

			childID := uuid.New()
			child, err := user.New(childID, parentID, time.Now())
			Expect(err).ToNot(HaveOccurred())

			Expect(userRepo.Save(ctx, child)).To(Succeed())
		})

		It("enforces single root constraint (idx_users_single_root)", func(ctx SpecContext) {
			root1, err := user.NewRoot(uuid.New(), time.Now())
			Expect(err).ToNot(HaveOccurred())
			Expect(userRepo.Save(ctx, root1)).To(Succeed())

			// Попытка создать второго пользователя без invited_by должна упасть
			root2, err := user.NewRoot(uuid.New(), time.Now())
			Expect(err).ToNot(HaveOccurred())

			Expect(userRepo.Save(ctx, root2)).To(HaveOccurred())
		})

		It("enforces foreign key constraint on invited_by", func(ctx SpecContext) {
			nonExistentParentID := uuid.New()
			child, err := user.New(uuid.New(), nonExistentParentID, time.Now())
			Expect(err).ToNot(HaveOccurred())

			Expect(userRepo.Save(ctx, child)).To(HaveOccurred())
		})
	})

	Describe("Descendants", func() {
		It("returns the target user and all their descendants recursively in the invitation tree", func(ctx SpecContext) {
			now := time.Now()

			// Построение дерева приглашений:
			// Root
			//  ├── Child 1
			//  │    └── Grandchild 1.1
			//  └── Child 2

			rootID := uuid.New()
			child1ID := uuid.New()
			child2ID := uuid.New()
			grandchild11ID := uuid.New()

			root, err := user.NewRoot(rootID, now)
			Expect(err).ToNot(HaveOccurred())
			child1, err := user.New(child1ID, rootID, now)
			Expect(err).ToNot(HaveOccurred())
			child2, err := user.New(child2ID, rootID, now)
			Expect(err).ToNot(HaveOccurred())
			grandchild11, err := user.New(grandchild11ID, child1ID, now)
			Expect(err).ToNot(HaveOccurred())

			Expect(userRepo.Save(ctx, root)).To(Succeed())
			Expect(userRepo.Save(ctx, child1)).To(Succeed())
			Expect(userRepo.Save(ctx, child2)).To(Succeed())
			Expect(userRepo.Save(ctx, grandchild11)).To(Succeed())

			// 1. Цепочка от Root -> Root + Child 1 + Child 2 + Grandchild 1.1
			rootChain, err := userRepo.Descendants(ctx, rootID)
			Expect(err).ToNot(HaveOccurred())
			Expect(rootChain).To(ConsistOf(rootID, child1ID, child2ID, grandchild11ID))

			// 2. Цепочка от Child 1 -> Child 1 + Grandchild 1.1
			child1Chain, err := userRepo.Descendants(ctx, child1ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(child1Chain).To(ConsistOf(child1ID, grandchild11ID))

			// 3. Цепочка от Child 2 (листовой узел) -> только Child 2
			child2Chain, err := userRepo.Descendants(ctx, child2ID)
			Expect(err).ToNot(HaveOccurred())
			Expect(child2Chain).To(Equal([]uuid.UUID{child2ID}))
		})

		It("returns an empty slice for non-existent user", func(ctx SpecContext) {
			descendants, err := userRepo.Descendants(ctx, uuid.New())
			Expect(err).To(MatchError(domain.ErrNotFound))
			Expect(descendants).To(BeEmpty())
		})
	})
	Describe("FindRoot", func() {
		It("successfully retrieves the root user when one exists", func(ctx SpecContext) {
			rootID := uuid.New()
			now := time.Now().Truncate(time.Microsecond)

			root, err := user.NewRoot(rootID, now)
			Expect(err).ToNot(HaveOccurred())
			Expect(userRepo.Save(ctx, root)).To(Succeed())

			childID := uuid.New()
			child, err := user.New(childID, rootID, now)
			Expect(err).ToNot(HaveOccurred())
			Expect(userRepo.Save(ctx, child)).To(Succeed())

			found, err := userRepo.FindRoot(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).ToNot(BeNil())

			Expect(found.ID()).To(Equal(rootID))
			Expect(found.InvitedBy()).To(BeNil())
			Expect(found.CreatedAt()).To(BeTemporally("~", now, time.Millisecond))
		})

		It("returns domain.ErrNotFound when no root user exists in the database", func(ctx SpecContext) {
			found, err := userRepo.FindRoot(ctx)
			Expect(err).To(MatchError(domain.ErrNotFound))
			Expect(found).To(BeNil())
		})
	})
})
