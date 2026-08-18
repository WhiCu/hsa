package application_test

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/mock"

	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/application/mocks"
	"github.com/whicu/hsa/internal/domain/invite"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("RootCreateInvite", func() {
	var (
		injector   do.Injector
		invites    *mocks.InviteSaver
		counter    *mocks.ActiveInviteCounter
		tokens     *mocks.TokenGenerator
		ids        *mocks.IDGenerator
		transactor *mocks.Transactor
		uc         *application.RootCreateInvite

		createdBy uuid.UUID
		ttl       time.Duration
	)

	BeforeEach(func() {
		invites = mocks.NewInviteSaver(GinkgoT())
		counter = mocks.NewActiveInviteCounter(GinkgoT())
		tokens = mocks.NewTokenGenerator(GinkgoT())
		ids = mocks.NewIDGenerator(GinkgoT())
		transactor = mocks.NewTransactor(GinkgoT())

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
				RootTTL:   ttl,
			},
		})

		var err error
		uc, err = do.Invoke[*application.RootCreateInvite](injector)
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("Execute", func() {
		It("successfully creates an invite even when root user has many active invites (UnlimitedPolicy)", func(ctx SpecContext) {
			expectedInviteID := uuid.New()
			rawToken := "raw_invite_token_123"
			codeHash := "sha256_hash_code_456"

			// Проксируем выполнение через транзакцию
			transactor.EXPECT().
				RunInTransaction(mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				}).
				Once()

			// Counter возвращает 9999 активных инвайтов — UnlimitedPolicy не должна возвращать ошибку
			counter.EXPECT().
				CountActiveByUser(mock.Anything, createdBy, mock.Anything).
				Return(9999, nil).
				Once()

			tokens.EXPECT().
				GenerateToken(32).
				Return(rawToken, codeHash, nil).
				Once()

			ids.EXPECT().
				NewID().
				Return(expectedInviteID).
				Once()

			invites.EXPECT().
				Save(mock.Anything, mock.MatchedBy(func(i *invite.Invite) bool {
					return i.ID() == expectedInviteID &&
						i.CreatedBy() == createdBy &&
						i.CodeHash() == codeHash &&
						i.UsedBy() == nil
				})).
				Return(nil).
				Once()

			resToken, expiresAt, err := uc.Execute(ctx, createdBy)

			Expect(err).NotTo(HaveOccurred())
			Expect(resToken).To(Equal(rawToken))
			Expect(expiresAt).To(BeTemporally("~", time.Now().Add(ttl), time.Second))
		})

		It("fails when active invite counter returns an error", func(ctx SpecContext) {
			counterErr := errors.New("failed to count active invites")

			transactor.EXPECT().
				RunInTransaction(mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				}).
				Once()

			counter.EXPECT().
				CountActiveByUser(mock.Anything, createdBy, mock.Anything).
				Return(0, counterErr).
				Once()

			resToken, expiresAt, err := uc.Execute(ctx, createdBy)

			Expect(err).To(MatchError(counterErr))
			Expect(resToken).To(BeEmpty())
			Expect(expiresAt).To(BeZero())
		})

		It("fails when token generator returns an error", func(ctx SpecContext) {
			tokenErr := errors.New("entropy source failed")

			transactor.EXPECT().
				RunInTransaction(mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				}).
				Once()

			counter.EXPECT().
				CountActiveByUser(mock.Anything, createdBy, mock.Anything).
				Return(0, nil).
				Once()

			tokens.EXPECT().
				GenerateToken(32).
				Return("", "", tokenErr).
				Once()

			resToken, expiresAt, err := uc.Execute(ctx, createdBy)

			Expect(err).To(MatchError(tokenErr))
			Expect(resToken).To(BeEmpty())
			Expect(expiresAt).To(BeZero())
		})

		It("fails when invite saver returns an error", func(ctx SpecContext) {
			saveErr := errors.New("database insert error")

			transactor.EXPECT().
				RunInTransaction(mock.Anything, mock.Anything).
				RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
					return fn(ctx)
				}).
				Once()

			counter.EXPECT().
				CountActiveByUser(mock.Anything, createdBy, mock.Anything).
				Return(5, nil).
				Once()

			tokens.EXPECT().
				GenerateToken(32).
				Return("token", "hash", nil).
				Once()

			ids.EXPECT().
				NewID().
				Return(uuid.New()).
				Once()

			invites.EXPECT().
				Save(mock.Anything, mock.Anything).
				Return(saveErr).
				Once()

			resToken, expiresAt, err := uc.Execute(ctx, createdBy)

			Expect(err).To(MatchError(saveErr))
			Expect(resToken).To(BeEmpty())
			Expect(expiresAt).To(BeZero())
		})
	})

	Describe("DI Container Resolution", func() {
		It("successfully resolves *RootCreateInvite from injector", func() {
			resolved, err := do.Invoke[*application.RootCreateInvite](injector)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolved).NotTo(BeNil())
		})
	})
})
