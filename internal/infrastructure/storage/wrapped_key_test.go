package storage_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"

	"github.com/whicu/hsa/internal/domain/credential"
	"github.com/whicu/hsa/internal/domain/key"
	"github.com/whicu/hsa/internal/domain/user"
	"github.com/whicu/hsa/internal/infrastructure/storage"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("WrappedKeyRepository", func() {
	var (
		injector       do.Injector
		st             *storage.Storage
		userRepo       *storage.UserRepository
		credRepo       *storage.CredentialRepository
		wrappedKeyRepo *storage.WrappedKeyRepository
		testUserID     uuid.UUID
		testCredID     uuid.UUID
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

		credRepo, err = do.Invoke[*storage.CredentialRepository](injector)
		Expect(err).ToNot(HaveOccurred())

		wrappedKeyRepo, err = do.Invoke[*storage.WrappedKeyRepository](injector)
		Expect(err).ToNot(HaveOccurred())

		Expect(st.Up(ctx)).To(Succeed())
		Expect(st.Reset(ctx)).To(Succeed())

		// Создаем базового пользователя и привязанный credential для удовлетворения FK
		testUserID = uuid.New()
		testUser, err := user.NewRoot(testUserID, time.Now())
		Expect(err).ToNot(HaveOccurred())
		Expect(userRepo.Save(ctx, testUser)).To(Succeed())

		testCredID = uuid.New()
		testCred := credential.Reconstruct(
			testCredID,
			[]byte("external_cred_for_keys_1"),
			testUserID,
			[]byte("public_key_bytes"),
			0,
			[]string{"internal"},
			time.Now(),
			nil,
		)
		Expect(credRepo.Save(ctx, testCred)).To(Succeed())
	})

	Describe("Save (CopyFrom Batch Insert)", func() {
		It("successfully saves multiple wrapped keys with different scopes (main and decoy)", func(ctx SpecContext) {
			now := time.Now().Truncate(time.Microsecond)

			mainKey := key.Reconstruct(
				uuid.New(),
				testUserID,
				testCredID,
				key.ScopeMain, // ENUM: 'main'
				[]byte("encrypted_main_dek_payload"),
				"AES-256-GCM-KW",
				now,
			)

			decoyKey := key.Reconstruct(
				uuid.New(),
				testUserID,
				testCredID,
				key.ScopeDecoy, // ENUM: 'decoy'
				[]byte("encrypted_decoy_dek_payload"),
				"AES-256-GCM-KW",
				now,
			)

			Expect(wrappedKeyRepo.Save(ctx, mainKey, decoyKey)).To(Succeed())
		})

		It("successfully handles empty keys slice without errors", func(ctx SpecContext) {
			Expect(wrappedKeyRepo.Save(ctx)).To(Succeed())
		})
	})

	Describe("Database Constraints Validation", func() {
		It("enforces unique constraint on (credential_id, scope)", func(ctx SpecContext) {
			now := time.Now()

			firstKey := key.Reconstruct(
				uuid.New(),
				testUserID,
				testCredID,
				key.ScopeMain,
				[]byte("first_payload"),
				"AES-256-GCM-KW",
				now,
			)
			Expect(wrappedKeyRepo.Save(ctx, firstKey)).To(Succeed())

			// Попытка сохранить второй ключ с тем же credential_id и тем же scope ('main')
			duplicateScopeKey := key.Reconstruct(
				uuid.New(),
				testUserID,
				testCredID,
				key.ScopeMain,
				[]byte("second_payload"),
				"AES-256-GCM-KW",
				now,
			)
			Expect(wrappedKeyRepo.Save(ctx, duplicateScopeKey)).To(HaveOccurred())
		})

		It("allows the same scope for different credentials", func(ctx SpecContext) {
			now := time.Now()

			// Создаем второй credential для того же пользователя
			secondCredID := uuid.New()
			secondCred := credential.Reconstruct(
				secondCredID,
				[]byte("external_cred_for_keys_2"),
				testUserID,
				[]byte("public_key_bytes_2"),
				0,
				[]string{"usb"},
				now,
				nil,
			)
			Expect(credRepo.Save(ctx, secondCred)).To(Succeed())

			key1 := key.Reconstruct(
				uuid.New(),
				testUserID,
				testCredID,
				key.ScopeMain,
				[]byte("payload_1"),
				"AES-256-GCM-KW",
				now,
			)
			key2 := key.Reconstruct(
				uuid.New(),
				testUserID,
				secondCredID,
				key.ScopeMain,
				[]byte("payload_2"),
				"AES-256-GCM-KW",
				now,
			)

			Expect(wrappedKeyRepo.Save(ctx, key1, key2)).To(Succeed())
		})

		It("enforces foreign key constraint on user_id", func(ctx SpecContext) {
			nonExistentUserID := uuid.New()
			orphanUserKey := key.Reconstruct(
				uuid.New(),
				nonExistentUserID,
				testCredID,
				key.ScopeMain,
				[]byte("payload"),
				"AES-256-GCM-KW",
				time.Now(),
			)

			Expect(wrappedKeyRepo.Save(ctx, orphanUserKey)).To(HaveOccurred())
		})

		It("enforces foreign key constraint on credential_id", func(ctx SpecContext) {
			nonExistentCredID := uuid.New()
			orphanCredKey := key.Reconstruct(
				uuid.New(),
				testUserID,
				nonExistentCredID,
				key.ScopeMain,
				[]byte("payload"),
				"AES-256-GCM-KW",
				time.Now(),
			)

			Expect(wrappedKeyRepo.Save(ctx, orphanCredKey)).To(HaveOccurred())
		})
	})
})
