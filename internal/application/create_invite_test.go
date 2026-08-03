package application_test

import (
	"context"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/mock"

	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/application/mocks"
	"github.com/whicu/hsa/internal/domain/invite"
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

		ctx       context.Context
		createdBy uuid.UUID
	)

	BeforeEach(func() {
		invites = mocks.NewInviteSaver(GinkgoT())
		counter = mocks.NewActiveInviteCounter(GinkgoT())
		tokens = mocks.NewTokenGenerator(GinkgoT())
		ids = mocks.NewIDGenerator(GinkgoT())
		policy = invite.NewPolicy(3) // limit = 3
		ttl = 24 * time.Hour

		uc = application.NewCreateInvite(invites, counter, tokens, ids, policy, ttl)

		ctx = context.Background()
		createdBy = uuid.New()
	})

	It("Basic Case: User has 0 active invites", func() {
		// Executing CreateInvite succeeds: creates Invite with codeHash, returns raw code once, sets ExpiresAt = now + ttl.
		counter.EXPECT().CountActiveByUser(ctx, createdBy, mock.AnythingOfType("time.Time")).Return(0, nil).Once()

		expectedCode := "secret-code"
		expectedHash := "hash"
		tokens.EXPECT().GenerateToken(32).Return(expectedCode, expectedHash, nil).Once()

		expectedID := uuid.New()
		ids.EXPECT().NewID().Return(expectedID).Once()

		invites.EXPECT().Save(ctx, mock.MatchedBy(func(inv *invite.Invite) bool {
			return inv.ID() == expectedID && inv.CreatedBy() == createdBy
		})).Return(nil).Once()

		code, expiresAt, err := uc.Execute(ctx, createdBy)

		Expect(err).ToNot(HaveOccurred())
		Expect(code).To(Equal(expectedCode))
		Expect(expiresAt).To(BeTemporally("~", time.Now().Add(ttl), time.Second))
	})

	It("Limit Reached: User has 3 active invites", func() {
		// Attempting to create a 4th fails with invite.ErrTooManyActive, nothing is saved.
		counter.EXPECT().CountActiveByUser(ctx, createdBy, mock.AnythingOfType("time.Time")).Return(3, nil).Once()

		code, expiresAt, err := uc.Execute(ctx, createdBy)

		Expect(err).To(MatchError(invite.ErrTooManyActive))
		Expect(code).To(BeEmpty())
		Expect(expiresAt.IsZero()).To(BeTrue())
	})

	It("Limit Freed by Expired or Redeemed Invites", func() {
		// The active count only counts active invites, so if the counter returns 2 (freed up space), it succeeds.
		counter.EXPECT().CountActiveByUser(ctx, createdBy, mock.AnythingOfType("time.Time")).Return(2, nil).Once()

		expectedCode := "secret-code-2"
		expectedHash := "hash-2"
		tokens.EXPECT().GenerateToken(32).Return(expectedCode, expectedHash, nil).Once()

		expectedID := uuid.New()
		ids.EXPECT().NewID().Return(expectedID).Once()

		invites.EXPECT().Save(ctx, mock.MatchedBy(func(inv *invite.Invite) bool {
			return inv.ID() == expectedID
		})).Return(nil).Once()

		code, _, err := uc.Execute(ctx, createdBy)

		Expect(err).ToNot(HaveOccurred())
		Expect(code).To(Equal(expectedCode))
	})
})
