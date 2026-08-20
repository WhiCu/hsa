package storage_test

import (
	"context"
	"net/netip"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"

	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/session"
	"github.com/whicu/hsa/internal/domain/user"
	"github.com/whicu/hsa/internal/infrastructure/storage"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("SessionRepository", func() {
	var (
		injector    do.Injector
		st          *storage.Storage
		userRepo    *storage.UserRepository
		sessionRepo *storage.SessionRepository
		testUserID  uuid.UUID
		testUser    *user.User
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

		sessionRepo, err = do.Invoke[*storage.SessionRepository](injector)
		Expect(err).ToNot(HaveOccurred())

		Expect(st.Up(ctx)).To(Succeed())
		Expect(st.Reset(ctx)).To(Succeed())

		// Создаем базового пользователя для удовлетворения Foreign Key (user_id)
		testUserID = uuid.New()
		testUser, err = user.NewRoot(testUserID, time.Now())
		Expect(err).ToNot(HaveOccurred())
		Expect(userRepo.Save(ctx, testUser)).To(Succeed())
	})

	Describe("Save & FindByTokenHash", func() {
		It("successfully saves and retrieves an active refresh token", func(ctx SpecContext) {
			tokenID := uuid.New()
			tokenHash := "sha256_active_token_hash_123"
			ip := netip.MustParseAddr("192.168.1.50")
			now := time.Now().Truncate(time.Microsecond)
			expiresAt := now.Add(30 * 24 * time.Hour)

			sess := session.Reconstruct(
				tokenID,
				testUserID,
				tokenHash,
				"Chrome / Windows 11",
				ip,
				expiresAt,
				nil, // revoked_at
				now,
			)

			Expect(sessionRepo.Save(ctx, sess)).To(Succeed())

			found, err := sessionRepo.FindByTokenHash(ctx, tokenHash)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).ToNot(BeNil())

			Expect(found.ID()).To(Equal(tokenID))
			Expect(found.UserID()).To(Equal(testUserID))
			Expect(found.TokenHash()).To(Equal(tokenHash))
			Expect(found.DeviceInfo()).To(Equal("Chrome / Windows 11"))
			Expect(found.IPAddress()).To(Equal(ip))
			Expect(found.ExpiresAt()).To(BeTemporally("~", expiresAt, time.Millisecond))
			Expect(found.RevokedAt()).To(BeNil())
			Expect(found.CreatedAt()).To(BeTemporally("~", now, time.Millisecond))
		})

		It("successfully updates revoked_at on conflict", func(ctx SpecContext) {
			tokenID := uuid.New()
			tokenHash := "sha256_token_to_revoke_456"
			ip := netip.MustParseAddr("10.0.0.1")
			now := time.Now().Truncate(time.Microsecond)
			expiresAt := now.Add(24 * time.Hour)

			// 1. Создаем активную сессию
			sess := session.Reconstruct(
				tokenID,
				testUserID,
				tokenHash,
				"Safari / iOS",
				ip,
				expiresAt,
				nil,
				now,
			)
			Expect(sessionRepo.Save(ctx, sess)).To(Succeed())

			// 2. Отзываем сессию (ON CONFLICT обновляет revoked_at)
			revokedAt := time.Now().Truncate(time.Microsecond)
			revokedSess := session.Reconstruct(
				tokenID,
				testUserID,
				tokenHash,
				"Safari / iOS",
				ip,
				expiresAt,
				&revokedAt,
				now,
			)
			Expect(sessionRepo.Save(ctx, revokedSess)).To(Succeed())

			// 3. Проверяем обновленное состояние
			found, err := sessionRepo.FindByTokenHash(ctx, tokenHash)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).ToNot(BeNil())
			Expect(found.RevokedAt()).ToNot(BeNil())
			Expect(*found.RevokedAt()).To(BeTemporally("~", revokedAt, time.Millisecond))
		})

		It("returns domain.ErrNotFound when querying a non-existent token hash", func(ctx SpecContext) {
			found, err := sessionRepo.FindByTokenHash(ctx, "non_existent_token_hash")
			Expect(err).To(MatchError(domain.ErrNotFound))
			Expect(found).To(BeNil())
		})
	})

	Describe("FindActiveByUserIDs", func() {
		It("returns active tokens only for requested users, filtering out expired and revoked ones", func(ctx SpecContext) {
			now := time.Now().Truncate(time.Microsecond)

			// Создаем второго пользователя
			secondUserID := uuid.New()
			secondUser, err := user.New(secondUserID, testUserID, now)
			Expect(err).ToNot(HaveOccurred())
			Expect(userRepo.Save(ctx, secondUser)).To(Succeed())

			// 1. Пользователь 1: активный токен -> ДОЛЖЕН вернуться (+1)
			t1 := session.Reconstruct(
				uuid.New(), testUserID, "hash_u1_active", "Desktop", netip.MustParseAddr("1.1.1.1"), now.Add(time.Hour), nil, now,
			)
			// 2. Пользователь 1: отозванный токен -> НЕ должен вернуться
			rev := now.Add(-5 * time.Minute)
			t2 := session.Reconstruct(
				uuid.New(), testUserID, "hash_u1_revoked", "Mobile", netip.MustParseAddr("1.1.1.1"), now.Add(time.Hour), &rev, now,
			)
			// 3. Пользователь 1: просроченный токен -> НЕ должен вернуться
			t3 := session.Reconstruct(
				uuid.New(), testUserID, "hash_u1_expired", "Tablet", netip.MustParseAddr("1.1.1.1"), now.Add(-time.Hour), nil, now.Add(-2*time.Hour),
			)
			// 4. Пользователь 2: активный токен -> ДОЛЖЕН вернуться (+1)
			t4 := session.Reconstruct(
				uuid.New(), secondUserID, "hash_u2_active", "Laptop", netip.MustParseAddr("2.2.2.2"), now.Add(time.Hour), nil, now,
			)

			// Сохраняем пакет токенов через Save (batchexec)
			Expect(sessionRepo.Save(ctx, t1, t2, t3, t4)).To(Succeed())

			// Запрашиваем активные токены для обоих пользователей
			activeTokens, err := sessionRepo.FindActiveByUserIDs(ctx, []user.UserID{testUserID, secondUserID}, now)
			Expect(err).ToNot(HaveOccurred())
			Expect(activeTokens).To(HaveLen(2))

			var activeIDs []uuid.UUID
			for _, t := range activeTokens {
				activeIDs = append(activeIDs, t.ID())
			}
			Expect(activeIDs).To(ConsistOf(t1.ID(), t4.ID()))

			// Запрашиваем активные токены только для первого пользователя
			u1Tokens, err := sessionRepo.FindActiveByUserIDs(ctx, []user.UserID{testUserID}, now)
			Expect(err).ToNot(HaveOccurred())
			Expect(u1Tokens).To(HaveLen(1))
			Expect(u1Tokens[0].ID()).To(Equal(t1.ID()))
		})

		It("returns empty slice when user has no active tokens or for empty list", func(ctx SpecContext) {
			tokens, err := sessionRepo.FindActiveByUserIDs(ctx, []user.UserID{uuid.New()}, time.Now())
			Expect(err).ToNot(HaveOccurred())
			Expect(tokens).To(BeEmpty())

			emptyTokens, err := sessionRepo.FindActiveByUserIDs(ctx, []user.UserID{}, time.Now())
			Expect(err).ToNot(HaveOccurred())
			Expect(emptyTokens).To(BeEmpty())
		})
	})
	Describe("FindByID", func() {
		It("successfully finds and reconstructs session by ID", func(ctx SpecContext) {
			tokenID := uuid.New()
			tokenHash := "sha256_hash_find_by_id_123"
			ip := netip.MustParseAddr("10.10.10.5")
			now := time.Now().Truncate(time.Microsecond)
			expiresAt := now.Add(7 * 24 * time.Hour)

			sess := session.Reconstruct(
				tokenID,
				testUserID,
				tokenHash,
				"Firefox / Linux",
				ip,
				expiresAt,
				nil,
				now,
			)
			Expect(sessionRepo.Save(ctx, sess)).To(Succeed())

			found, err := sessionRepo.FindByID(ctx, tokenID)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).ToNot(BeNil())

			Expect(found.ID()).To(Equal(tokenID))
			Expect(found.UserID()).To(Equal(testUserID))
			Expect(found.TokenHash()).To(Equal(tokenHash))
			Expect(found.DeviceInfo()).To(Equal("Firefox / Linux"))
			Expect(found.IPAddress()).To(Equal(ip))
			Expect(found.ExpiresAt()).To(BeTemporally("~", expiresAt, time.Millisecond))
			Expect(found.RevokedAt()).To(BeNil())
			Expect(found.CreatedAt()).To(BeTemporally("~", now, time.Millisecond))
		})

		It("successfully retrieves a revoked session by ID", func(ctx SpecContext) {
			tokenID := uuid.New()
			tokenHash := "sha256_hash_revoked_find_by_id"
			ip := netip.MustParseAddr("10.10.10.6")
			now := time.Now().Truncate(time.Microsecond)
			expiresAt := now.Add(24 * time.Hour)
			revokedAt := now.Add(time.Hour)

			sess := session.Reconstruct(
				tokenID,
				testUserID,
				tokenHash,
				"Chrome / Android",
				ip,
				expiresAt,
				&revokedAt,
				now,
			)
			Expect(sessionRepo.Save(ctx, sess)).To(Succeed())

			found, err := sessionRepo.FindByID(ctx, tokenID)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).ToNot(BeNil())
			Expect(found.RevokedAt()).ToNot(BeNil())
			Expect(*found.RevokedAt()).To(BeTemporally("~", revokedAt, time.Millisecond))
		})

		It("returns domain.ErrNotFound when session ID does not exist", func(ctx SpecContext) {
			found, err := sessionRepo.FindByID(ctx, uuid.New())
			Expect(err).To(MatchError(domain.ErrNotFound))
			Expect(found).To(BeNil())
		})
	})

	Describe("Database Constraints Validation", func() {
		It("enforces unique constraint on token_hash", func(ctx SpecContext) {
			sharedHash := "duplicate_token_hash_val"
			now := time.Now()

			s1 := session.Reconstruct(
				uuid.New(), testUserID, sharedHash, "Dev1", netip.MustParseAddr("1.1.1.1"), now.Add(time.Hour), nil, now,
			)
			Expect(sessionRepo.Save(ctx, s1)).To(Succeed())

			// Попытка сохранить второй токен с другим ID, но тем же token_hash, должна вернуть ошибку
			s2 := session.Reconstruct(
				uuid.New(), testUserID, sharedHash, "Dev2", netip.MustParseAddr("1.1.1.1"), now.Add(time.Hour), nil, now,
			)
			Expect(sessionRepo.Save(ctx, s2)).To(HaveOccurred())
		})

		It("enforces foreign key constraint on user_id", func(ctx SpecContext) {
			nonExistentUserID := uuid.New()
			orphanSession := session.Reconstruct(
				uuid.New(), nonExistentUserID, "orphan_hash_val", "Dev", netip.MustParseAddr("1.1.1.1"), time.Now().Add(time.Hour), nil, time.Now(),
			)

			Expect(sessionRepo.Save(ctx, orphanSession)).To(HaveOccurred())
		})
	})
})
