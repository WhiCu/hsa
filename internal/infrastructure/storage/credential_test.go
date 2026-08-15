package storage_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"

	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/credential"
	"github.com/whicu/hsa/internal/domain/user"
	"github.com/whicu/hsa/internal/infrastructure/storage"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("CredentialRepository", func() {
	usbString := "usb"
	var (
		injector   do.Injector
		st         *storage.Storage
		userRepo   *storage.UserRepository
		credRepo   *storage.CredentialRepository
		testUser   *user.User
		testUserID uuid.UUID
	)

	BeforeEach(func(ctx SpecContext) {
		// 1. Инициализируем DI-контейнер
		injector = do.New(storage.Package)

		do.OverrideValue(injector, logger.NewNOPSlog())
		do.OverrideValue(injector, globalConfig)
		do.OverrideValue(injector, context.Background())

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

		credRepo, err = do.Invoke[*storage.CredentialRepository](injector)
		Expect(err).ToNot(HaveOccurred())

		// 3. Схема и очистка БД
		Expect(st.Up(ctx)).To(Succeed())
		Expect(st.Reset(ctx)).To(Succeed())

		// 4. Создаем базового пользователя для привязки credentials
		testUserID = uuid.New()
		testUser, err = user.NewRoot(testUserID, time.Now())
		Expect(err).ToNot(HaveOccurred())
		Expect(userRepo.Save(ctx, testUser)).To(Succeed())
	})

	Describe("Save & FindByExternalID", func() {
		It("successfully saves and retrieves an active credential", func(ctx SpecContext) {
			credID := uuid.New()
			externalID := []byte("fido2-external-id-12345")
			pubKey := []byte("raw-public-key-bytes")
			transports := []string{usbString, "nfc", "ble"}
			now := time.Now().Truncate(time.Microsecond)

			cred := credential.Reconstruct(
				credID,
				externalID,
				testUserID,
				pubKey,
				10, // sign_count
				transports,
				now,
				nil, // revoked_at (активный ключ)
			)

			// Сохраняем
			Expect(credRepo.Save(ctx, cred)).To(Succeed())

			// Получаем по ExternalID
			found, err := credRepo.FindByExternalID(ctx, externalID)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).ToNot(BeNil())

			Expect(found.ID()).To(Equal(credID))
			Expect(found.ExternalID()).To(Equal(externalID))
			Expect(found.UserID()).To(Equal(testUserID))
			Expect(found.PublicKey()).To(Equal(pubKey))
			Expect(found.SignCount()).To(Equal(uint32(10)))
			Expect(found.Transports()).To(ConsistOf(usbString, "nfc", "ble"))
			Expect(found.CreatedAt()).To(BeTemporally("~", now, time.Millisecond))
			Expect(found.RevokedAt()).To(BeNil())
		})

		It("successfully saves and retrieves a revoked credential", func(ctx SpecContext) {
			credID := uuid.New()
			externalID := []byte("fido2-revoked-id-67890")
			pubKey := []byte("raw-public-key-bytes")
			createdAt := time.Now().Add(-1 * time.Hour).Truncate(time.Microsecond)
			revokedAt := time.Now().Truncate(time.Microsecond)

			cred := credential.Reconstruct(
				credID,
				externalID,
				testUserID,
				pubKey,
				42,
				[]string{"internal"},
				createdAt,
				&revokedAt,
			)

			Expect(credRepo.Save(ctx, cred)).To(Succeed())

			found, err := credRepo.FindByExternalID(ctx, externalID)
			Expect(err).ToNot(HaveOccurred())
			Expect(found).ToNot(BeNil())
			Expect(found.RevokedAt()).ToNot(BeNil())
			Expect(*found.RevokedAt()).To(BeTemporally("~", revokedAt, time.Millisecond))
		})

		It("returns domain.ErrNotFound when querying a non-existent external ID", func(ctx SpecContext) {
			found, err := credRepo.FindByExternalID(ctx, []byte("unknown-external-id"))
			Expect(err).To(MatchError(domain.ErrNotFound))
			Expect(found).To(BeNil())
		})
	})

	Describe("FindByUserID", func() {
		It("returns all credentials belonging to the user", func(ctx SpecContext) {
			now := time.Now().Truncate(time.Microsecond)

			// Создаем 2 ключа для основного пользователя
			cred1 := credential.Reconstruct(
				uuid.New(), []byte("key-1"), testUserID, []byte("pk1"), 1, []string{usbString}, now, nil,
			)
			cred2 := credential.Reconstruct(
				uuid.New(), []byte("key-2"), testUserID, []byte("pk2"), 5, []string{"hybrid"}, now, nil,
			)

			// Создаем второго пользователя и его собственный ключ
			otherUserID := uuid.New()
			otherUser, err := user.New(otherUserID, testUserID, now)
			Expect(err).ToNot(HaveOccurred())
			Expect(userRepo.Save(ctx, otherUser)).To(Succeed())

			otherCred := credential.Reconstruct(
				uuid.New(), []byte("key-other"), otherUserID, []byte("pk-other"), 0, []string{"nfc"}, now, nil,
			)

			Expect(credRepo.Save(ctx, cred1)).To(Succeed())
			Expect(credRepo.Save(ctx, cred2)).To(Succeed())
			Expect(credRepo.Save(ctx, otherCred)).To(Succeed())

			// 1. Поиск для testUserID -> должен вернуть ровно 2 ключа
			userCreds, err := credRepo.FindByUserID(ctx, testUserID)
			Expect(err).ToNot(HaveOccurred())
			Expect(userCreds).To(HaveLen(2))

			var ids []uuid.UUID
			for _, c := range userCreds {
				ids = append(ids, c.ID())
			}
			Expect(ids).To(ConsistOf(cred1.ID(), cred2.ID()))

			// 2. Поиск для otherUserID -> должен вернуть ровно 1 ключ
			otherCreds, err := credRepo.FindByUserID(ctx, otherUserID)
			Expect(err).ToNot(HaveOccurred())
			Expect(otherCreds).To(HaveLen(1))
			Expect(otherCreds[0].ID()).To(Equal(otherCred.ID()))
		})

		It("returns an empty slice when user has no credentials", func(ctx SpecContext) {
			creds, err := credRepo.FindByUserID(ctx, testUserID)
			Expect(err).ToNot(HaveOccurred())
			Expect(creds).To(BeEmpty())
		})

		It("returns an empty slice for non-existent user ID", func(ctx SpecContext) {
			creds, err := credRepo.FindByUserID(ctx, uuid.New())
			Expect(err).ToNot(HaveOccurred())
			Expect(creds).To(BeEmpty())
		})
	})

	Describe("Database Constraints Validation", func() {
		It("enforces unique constraint on external_id", func(ctx SpecContext) {
			sharedExternalID := []byte("duplicate-external-id")
			now := time.Now()

			cred1 := credential.Reconstruct(
				uuid.New(), sharedExternalID, testUserID, []byte("pk1"), 0, []string{}, now, nil,
			)
			Expect(credRepo.Save(ctx, cred1)).To(Succeed())

			// Попытка сохранить второй ключ с тем же external_id должна упасть
			cred2 := credential.Reconstruct(
				uuid.New(), sharedExternalID, testUserID, []byte("pk2"), 0, []string{}, now, nil,
			)
			Expect(credRepo.Save(ctx, cred2)).To(HaveOccurred())
		})

		It("enforces foreign key constraint on user_id", func(ctx SpecContext) {
			nonExistentUserID := uuid.New()
			cred := credential.Reconstruct(
				uuid.New(), []byte("orphan-key"), nonExistentUserID, []byte("pk"), 0, []string{}, time.Now(), nil,
			)

			// Падает из-за FK REFERENCES users(id)
			Expect(credRepo.Save(ctx, cred)).To(HaveOccurred())
		})
	})
})
