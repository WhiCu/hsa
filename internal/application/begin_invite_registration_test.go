package application_test

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/application/mocks"
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/invite"
)

var _ = Describe("BeginInviteRegistration", func() {
	var (
		invites     *mocks.InviteFinderByCode
		ids         *mocks.IDGenerator
		registrator *mocks.Registrator
		hashGen     *mocks.HashGenerator
		uc          *application.BeginInviteRegistration

		ctx        context.Context
		inviteCode string
		hashResult string
	)

	BeforeEach(func() {
		invites = mocks.NewInviteFinderByCode(GinkgoT())
		ids = mocks.NewIDGenerator(GinkgoT())
		registrator = mocks.NewRegistrator(GinkgoT())
		hashGen = mocks.NewHashGenerator(GinkgoT())

		uc = application.NewBeginInviteRegistration(invites, ids, registrator, hashGen)

		ctx = context.Background()
		inviteCode = "raw-invite-code"
		hashResult = "hashed-invite-code"
	})

	It("Basic Case: Valid, non-expired, unredeemed code", func() {
		hashGen.EXPECT().GenerateHash(inviteCode).Return(hashResult, nil).Once()

		invID := uuid.New()
		createdBy := uuid.New()
		inv, _ := invite.New(invID, createdBy, hashResult, 24*time.Hour, time.Now())

		invites.EXPECT().FindByCodeHash(ctx, hashResult).Return(inv, nil).Once()

		candidateUserID := uuid.New()
		ids.EXPECT().NewID().Return(candidateUserID).Once()

		expectedToken := "challenge-token"
		expectedOpts := []byte("creation-options")
		registrator.EXPECT().Begin(ctx, candidateUserID, invID).Return(expectedToken, expectedOpts, nil).Once()

		token, opts, err := uc.Execute(ctx, inviteCode)

		Expect(err).ToNot(HaveOccurred())
		Expect(token).To(Equal(expectedToken))
		Expect(opts).To(Equal(expectedOpts))
	})

	It("Non-existent Code", func() {
		hashGen.EXPECT().GenerateHash(inviteCode).Return(hashResult, nil).Once()
		invites.EXPECT().FindByCodeHash(ctx, hashResult).Return(nil, domain.ErrNotFound).Once()

		token, opts, err := uc.Execute(ctx, inviteCode)

		Expect(err).To(MatchError(domain.ErrNotFound))
		Expect(token).To(BeEmpty())
		Expect(opts).To(BeNil())
	})

	It("Expired Invite", func() {
		hashGen.EXPECT().GenerateHash(inviteCode).Return(hashResult, nil).Once()

		invID := uuid.New()
		createdBy := uuid.New()
		// creation time 48 hours ago, ttl 24 hours -> expired
		inv, _ := invite.New(invID, createdBy, hashResult, 24*time.Hour, time.Now().Add(-48*time.Hour))

		invites.EXPECT().FindByCodeHash(ctx, hashResult).Return(inv, nil).Once()

		token, opts, err := uc.Execute(ctx, inviteCode)

		Expect(err).To(MatchError(invite.ErrExpired))
		Expect(token).To(BeEmpty())
		Expect(opts).To(BeNil())
	})

	It("Already Used Invite", func() {
		hashGen.EXPECT().GenerateHash(inviteCode).Return(hashResult, nil).Once()

		invID := uuid.New()
		createdBy := uuid.New()
		inv, _ := invite.New(invID, createdBy, hashResult, 24*time.Hour, time.Now())
		inv.Redeem(uuid.New(), time.Now())

		invites.EXPECT().FindByCodeHash(ctx, hashResult).Return(inv, nil).Once()

		token, opts, err := uc.Execute(ctx, inviteCode)

		Expect(err).To(MatchError(invite.ErrAlreadyUsed))
		Expect(token).To(BeEmpty())
		Expect(opts).To(BeNil())
	})

	It("Hash Generation Failure", func() {
		expectedErr := errors.New("hash generation failed")
		hashGen.EXPECT().GenerateHash(inviteCode).Return("", expectedErr).Once()

		token, opts, err := uc.Execute(ctx, inviteCode)

		Expect(err).To(MatchError(expectedErr))
		Expect(token).To(BeEmpty())
		Expect(opts).To(BeNil())
	})

	It("Registrator Begin Failure", func() {
		hashGen.EXPECT().GenerateHash(inviteCode).Return(hashResult, nil).Once()

		invID := uuid.New()
		createdBy := uuid.New()
		inv, _ := invite.New(invID, createdBy, hashResult, 24*time.Hour, time.Now())

		invites.EXPECT().FindByCodeHash(ctx, hashResult).Return(inv, nil).Once()

		candidateUserID := uuid.New()
		ids.EXPECT().NewID().Return(candidateUserID).Once()

		expectedErr := errors.New("registrator begin failed")
		registrator.EXPECT().Begin(ctx, candidateUserID, invID).Return("", nil, expectedErr).Once()

		token, opts, err := uc.Execute(ctx, inviteCode)

		Expect(err).To(MatchError(expectedErr))
		Expect(token).To(BeEmpty())
		Expect(opts).To(BeNil())
	})
})
