package application_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/application/mocks"
	"github.com/whicu/hsa/internal/domain/invite"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("CreateInvite", func() {
	var (
		invites *mocks.InviteSaver
		counter *mocks.ActiveInviteCounter
		tokens  *mocks.TokenGenerator
		ids     *mocks.IDGenerator
		policy  invite.Policy
		ttl     time.Duration
		uc      *application.CreateInvite

		createdBy uuid.UUID
	)

	BeforeEach(func() {
		invites = mocks.NewInviteSaver(GinkgoT())
		counter = mocks.NewActiveInviteCounter(GinkgoT())
		tokens = mocks.NewTokenGenerator(GinkgoT())
		ids = mocks.NewIDGenerator(GinkgoT())
		policy = invite.NewPolicy(3) // limit = 3
		ttl = 24 * time.Hour

		uc = application.NewCreateInvite(logger.NewNOPSlog(), invites, counter, tokens, ids, policy, ttl)

		createdBy = uuid.New()
	})

	// --- Helper Functions ---

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
		It("Fails when ActiveInviteCounter returns a database error", func(ctx SpecContext) {
			synctest.Test(testT, func(_ *testing.T) {
				now := time.Now()
				dbErr := errors.New("db failure: failed to count active invites")

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
})
