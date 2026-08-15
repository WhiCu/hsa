package storage_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"

	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/invite"
	"github.com/whicu/hsa/internal/domain/user"
	"github.com/whicu/hsa/internal/infrastructure/storage"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("InviteRepository", func() {
	var (
		injector      do.Injector
		st            *storage.Storage
		userRepo      *storage.UserRepository
		inviteRepo    *storage.InviteRepository
		creatorUser   *user.User
		creatorUserID uuid.UUID
		usedByUser    *user.User
		usedByUserID  uuid.UUID
	)

	BeforeEach(func(ctx SpecContext) {
		injector = do.New(storage.Package)

		do.OverrideValue(injector, context.Background())
		do.OverrideValue(injector, logger.NewNOPSlog())
		do.OverrideValue(injector, globalConfig)

		var err error
		st, err = do.Invoke[*storage.Storage](injector)
		Expect(err).ToNot(HaveOccurred())
		DeferCleanup(func(_ SpecContext) {
			rep := injector.Shutdown()
			Expect(rep.Succeed).To(BeTrue())
		})

		userRepo, err = do.Invoke[*storage.UserRepository](injector)
		Expect(err).ToNot(HaveOccurred())

		inviteRepo, err = do.Invoke[*storage.InviteRepository](injector)
		Expect(err).ToNot(HaveOccurred())

		Expect(st.Up(ctx)).To(Succeed())
		Expect(st.Reset(ctx)).To(Succeed())

		// Создаем пользователей для соблюдения Foreign Key (created_by / used_by)
		creatorUserID = uuid.New()
		creatorUser, err = user.NewRoot(creatorUserID, time.Now())
		Expect(err).ToNot(HaveOccurred())
		Expect(userRepo.Save(ctx, creatorUser)).To(Succeed())

		usedByUserID = uuid.New()
		usedByUser, err = user.New(usedByUserID, creatorUserID, time.Now())
		Expect(err).ToNot(HaveOccurred())
		Expect(userRepo.Save(ctx, usedByUser)).To(Succeed())
	})

	Describe("Save & FindByID", func() {
		It("successfully saves and retrieves an active unused invite", func(ctx SpecContext) {
			inviteID := uuid.New()
			codeHash := "hash_active_invite_123"
			now := time.Now().Truncate(time.Microsecond)
			expiresAt := now.Add(24 * time.Hour)

			inv := invite.Reconstruct(
				inviteID,
				creatorUserID,
				codeHash,
				nil, // used_by
				nil, // used_at
				expiresAt,
				now,
			)

			Expect(inviteRepo.Save(ctx, inv)).To(Succeed())

			found, err := inviteRepo.FindByID(ctx, inviteID)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).ToNot(BeNil())

			Expect(found.ID()).To(Equal(inviteID))
			Expect(found.CreatedBy()).To(Equal(creatorUserID))
			Expect(found.CodeHash()).To(Equal(codeHash))
			Expect(found.UsedBy()).To(BeNil())
			Expect(found.UsedAt()).To(BeNil())
			Expect(found.ExpiresAt()).To(BeTemporally("~", expiresAt, time.Millisecond))
			Expect(found.CreatedAt()).To(BeTemporally("~", now, time.Millisecond))
		})

		It("successfully saves and retrieves an invite marked as used", func(ctx SpecContext) {
			inviteID := uuid.New()
			codeHash := "hash_used_invite_456"
			now := time.Now().Truncate(time.Microsecond)
			usedAt := now.Add(1 * time.Hour)
			expiresAt := now.Add(24 * time.Hour)

			inv := invite.Reconstruct(
				inviteID,
				creatorUserID,
				codeHash,
				&usedByUserID,
				&usedAt,
				expiresAt,
				now,
			)

			Expect(inviteRepo.Save(ctx, inv)).To(Succeed())

			found, err := inviteRepo.FindByID(ctx, inviteID)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).ToNot(BeNil())
			Expect(found.UsedBy()).ToNot(BeNil())
			Expect(*found.UsedBy()).To(Equal(usedByUserID))
			Expect(found.UsedAt()).ToNot(BeNil())
			Expect(*found.UsedAt()).To(BeTemporally("~", usedAt, time.Millisecond))
		})

		It("returns domain.ErrNotFound when querying a non-existent invite ID", func(ctx SpecContext) {
			found, err := inviteRepo.FindByID(ctx, uuid.New())
			Expect(err).To(MatchError(domain.ErrNotFound))
			Expect(found).To(BeNil())
		})
	})

	Describe("FindByCodeHash", func() {
		It("successfully finds an invite by its code hash", func(ctx SpecContext) {
			inviteID := uuid.New()
			codeHash := "unique_code_hash_789"
			now := time.Now().Truncate(time.Microsecond)

			inv := invite.Reconstruct(
				inviteID,
				creatorUserID,
				codeHash,
				nil,
				nil,
				now.Add(time.Hour),
				now,
			)
			Expect(inviteRepo.Save(ctx, inv)).To(Succeed())

			found, err := inviteRepo.FindByCodeHash(ctx, codeHash)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).ToNot(BeNil())
			Expect(found.ID()).To(Equal(inviteID))
			Expect(found.CodeHash()).To(Equal(codeHash))
		})

		It("returns domain.ErrNotFound when querying a non-existent code hash", func(ctx SpecContext) {
			found, err := inviteRepo.FindByCodeHash(ctx, "non_existent_hash")
			Expect(err).To(MatchError(domain.ErrNotFound))
			Expect(found).To(BeNil())
		})
	})

	Describe("CountActiveByUser", func() {
		It("counts only active, unexpired and unused invites created by the user", func(ctx SpecContext) {
			now := time.Now().Truncate(time.Microsecond)
			usedAt := now.Add(-10 * time.Minute)

			// 1. Активный валидный инвайт (used_by = nil, used_at = nil, expires_at > now) -> ДОЛЖЕН учитываться (+1)
			activeInv1 := invite.Reconstruct(
				uuid.New(), creatorUserID, "active_hash_1", nil, nil, now.Add(2*time.Hour), now,
			)

			// 2. Второй активный валидный инвайт -> ДОЛЖЕН учитываться (+1)
			activeInv2 := invite.Reconstruct(
				uuid.New(), creatorUserID, "active_hash_2", nil, nil, now.Add(5*time.Hour), now,
			)

			// 3. Просроченный инвайт (expires_at < now) -> НЕ должен учитываться
			expiredInv := invite.Reconstruct(
				uuid.New(), creatorUserID, "expired_hash", nil, nil, now.Add(-1*time.Hour), now.Add(-2*time.Hour),
			)

			// 4. Погашенный инвайт (used_by != nil, used_at != nil) -> НЕ должен учитываться
			usedInv := invite.Reconstruct(
				uuid.New(), creatorUserID, "used_hash", &usedByUserID, &usedAt, now.Add(2*time.Hour), now,
			)

			// 5. Активный инвайт другого пользователя -> НЕ должен учитываться для creatorUserID
			otherUserInv := invite.Reconstruct(
				uuid.New(), usedByUserID, "other_user_hash", nil, nil, now.Add(2*time.Hour), now,
			)

			Expect(inviteRepo.Save(ctx, activeInv1)).To(Succeed())
			Expect(inviteRepo.Save(ctx, activeInv2)).To(Succeed())
			Expect(inviteRepo.Save(ctx, expiredInv)).To(Succeed())
			Expect(inviteRepo.Save(ctx, usedInv)).To(Succeed())
			Expect(inviteRepo.Save(ctx, otherUserInv)).To(Succeed())

			// Проверяем количество активных для creatorUserID
			count, err := inviteRepo.CountActiveByUser(ctx, creatorUserID, now)
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(2))

			// Проверяем количество активных для usedByUserID
			otherCount, err := inviteRepo.CountActiveByUser(ctx, usedByUserID, now)
			Expect(err).ToNot(HaveOccurred())
			Expect(otherCount).To(Equal(1))
		})

		It("returns 0 when user has no invites", func(ctx SpecContext) {
			count, err := inviteRepo.CountActiveByUser(ctx, creatorUserID, time.Now())
			Expect(err).ToNot(HaveOccurred())
			Expect(count).To(Equal(0))
		})
	})

	Describe("Database Constraints Validation", func() {
		It("enforces unique constraint on code_hash", func(ctx SpecContext) {
			sharedHash := "duplicate_hash_value"
			now := time.Now()

			inv1 := invite.Reconstruct(
				uuid.New(), creatorUserID, sharedHash, nil, nil, now.Add(time.Hour), now,
			)
			Expect(inviteRepo.Save(ctx, inv1)).To(Succeed())

			// Попытка сохранить второй инвайт с тем же code_hash должна вызвать ошибку уникальности
			inv2 := invite.Reconstruct(
				uuid.New(), creatorUserID, sharedHash, nil, nil, now.Add(time.Hour), now,
			)
			Expect(inviteRepo.Save(ctx, inv2)).To(HaveOccurred())
		})

		It("enforces foreign key constraint on created_by", func(ctx SpecContext) {
			nonExistentUserID := uuid.New()
			inv := invite.Reconstruct(
				uuid.New(), nonExistentUserID, "orphan_hash", nil, nil, time.Now().Add(time.Hour), time.Now(),
			)

			Expect(inviteRepo.Save(ctx, inv)).To(HaveOccurred())
		})

		It("enforces foreign key constraint on used_by", func(ctx SpecContext) {
			nonExistentUserID := uuid.New()
			usedAt := time.Now()
			inv := invite.Reconstruct(
				uuid.New(), creatorUserID, "used_orphan_hash", &nonExistentUserID, &usedAt, time.Now().Add(time.Hour), time.Now(),
			)

			Expect(inviteRepo.Save(ctx, inv)).To(HaveOccurred())
		})
	})
})
