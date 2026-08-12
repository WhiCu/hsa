package application_test

import (
	"errors"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"

	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/application/mocks"
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/invite"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("BeginInviteRegistration", func() {
	var (
		injector    do.Injector
		invites     *mocks.InviteFinderByCode
		ids         *mocks.IDGenerator
		registrator *mocks.Registrator
		hashGen     *mocks.HashGenerator
		uc          *application.BeginInviteRegistration

		inviteCode string
		hashResult string
	)

	BeforeEach(func() {
		{
			invites = mocks.NewInviteFinderByCode(GinkgoT())
			ids = mocks.NewIDGenerator(GinkgoT())
			registrator = mocks.NewRegistrator(GinkgoT())
			hashGen = mocks.NewHashGenerator(GinkgoT())
		}
		{
			injector = do.New(application.Package)

			do.OverrideValue[application.InviteFinderByCode](injector, invites)
			do.OverrideValue[application.IDGenerator](injector, ids)
			do.OverrideValue[application.Registrator](injector, registrator)
			do.OverrideValue[application.HashGenerator](injector, hashGen)

			do.OverrideValue(injector, logger.NewNOPSlog())

			var err error
			uc, err = do.Invoke[*application.BeginInviteRegistration](injector)
			Expect(err).ToNot(HaveOccurred())
		}

		inviteCode = "raw-invite-code"
		hashResult = "hashed-invite-code"
	})

	expectHashOK := func() {
		hashGen.EXPECT().GenerateHash(inviteCode).Return(hashResult, nil).Once()
	}

	newInvite := func(ttl time.Duration, createdAt time.Time) *invite.Invite {
		invID := uuid.New()
		createdBy := uuid.New()
		inv, err := invite.New(invID, createdBy, hashResult, ttl, createdAt)
		Expect(err).ToNot(HaveOccurred())
		return inv
	}

	It("Basic Case: Valid, non-expired, unredeemed code", func(ctx SpecContext) {
		expectHashOK()

		inv := newInvite(24*time.Hour, time.Now())
		invites.EXPECT().FindByCodeHash(ctx, hashResult).Return(inv, nil).Once()

		candidateUserID := uuid.New()
		ids.EXPECT().NewID().Return(candidateUserID).Once()

		expectedToken := testChallengeToken
		expectedOpts := []byte("creation-options")
		registrator.EXPECT().Begin(ctx, candidateUserID, inv.ID()).Return(expectedToken, expectedOpts, nil).Once()

		token, opts, err := uc.Execute(ctx, inviteCode)

		Expect(err).ToNot(HaveOccurred())
		Expect(token).To(Equal(expectedToken))
		Expect(opts).To(Equal(expectedOpts))
	})

	DescribeTable("Invite lookup fails",
		func(ctx SpecContext, buildResult func() (*invite.Invite, error), expectedErr error) {
			expectHashOK()

			inv, findErr := buildResult()
			invites.EXPECT().FindByCodeHash(ctx, hashResult).Return(inv, findErr).Once()

			token, opts, err := uc.Execute(ctx, inviteCode)

			Expect(err).To(MatchError(expectedErr))
			Expect(token).To(BeEmpty())
			Expect(opts).To(BeNil())
		},
		Entry("Non-existent Code",
			func() (*invite.Invite, error) { return nil, domain.ErrNotFound },
			domain.ErrNotFound,
		),
		Entry("Expired Invite",
			func() (*invite.Invite, error) {
				inv := newInvite(24*time.Hour, time.Now().Add(-48*time.Hour))
				return inv, nil
			},
			invite.ErrExpired,
		),
		Entry("Already Used Invite",
			func() (*invite.Invite, error) {
				inv := newInvite(24*time.Hour, time.Now())
				inv.Redeem(uuid.New(), time.Now())
				return inv, nil
			},
			invite.ErrAlreadyUsed,
		),
	)

	It("Hash Generation Failure", func(ctx SpecContext) {
		expectedErr := errors.New("hash generation failed")
		hashGen.EXPECT().GenerateHash(inviteCode).Return("", expectedErr).Once()

		token, opts, err := uc.Execute(ctx, inviteCode)

		Expect(err).To(MatchError(expectedErr))
		Expect(token).To(BeEmpty())
		Expect(opts).To(BeNil())
	})

	It("Registrator Begin Failure", func(ctx SpecContext) {
		expectHashOK()

		inv := newInvite(24*time.Hour, time.Now())
		invites.EXPECT().FindByCodeHash(ctx, hashResult).Return(inv, nil).Once()

		candidateUserID := uuid.New()
		ids.EXPECT().NewID().Return(candidateUserID).Once()

		expectedErr := errors.New("registrator begin failed")
		registrator.EXPECT().Begin(ctx, candidateUserID, inv.ID()).Return("", nil, expectedErr).Once()

		token, opts, err := uc.Execute(ctx, inviteCode)

		Expect(err).To(MatchError(expectedErr))
		Expect(token).To(BeEmpty())
		Expect(opts).To(BeNil())
	})
})
