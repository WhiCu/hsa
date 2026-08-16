package application_test

import (
	"errors"
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"
	"github.com/stretchr/testify/mock"

	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/application/mocks"
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/user"
	"github.com/whicu/hsa/pkg/logger"
)

var _ = Describe("Bootstrap", func() {
	var (
		injector do.Injector
		finder   *mocks.RootFinder
		saver    *mocks.UserSaver
		idGen    *mocks.IDGenerator
		uc       *application.BootstrapRoot
	)

	BeforeEach(func() {
		{
			finder = mocks.NewRootFinder(GinkgoT())
			saver = mocks.NewUserSaver(GinkgoT())
			idGen = mocks.NewIDGenerator(GinkgoT())
		}
		{
			injector = do.New(application.Package)

			do.OverrideValue(injector, finder)
			do.OverrideValue(injector, saver)
			do.OverrideValue(injector, idGen)

			do.OverrideValue(injector, logger.NewNOPSlog())

			var err error
			uc, err = do.Invoke[*application.BootstrapRoot](injector)
			Expect(err).ToNot(HaveOccurred())
		}

	})

	It("successfully creates and saves a root user when no root exists", func(ctx SpecContext) {
		expectedRootID := uuid.New()

		finder.EXPECT().
			FindRoot(mock.Anything).
			Return(nil, domain.ErrNotFound).
			Once()

		idGen.EXPECT().
			NewID().
			Return(expectedRootID).
			Once()

		saver.EXPECT().
			Save(mock.Anything, mock.MatchedBy(func(u *user.User) bool {
				return u.ID() == expectedRootID && u.InvitedBy() == nil
			})).
			Return(nil).
			Once()

		root, err := uc.Execute(ctx)

		Expect(err).NotTo(HaveOccurred())
		Expect(root).NotTo(BeNil())
		Expect(root.ID()).To(Equal(expectedRootID))
		Expect(root.InvitedBy()).To(BeNil())
	})

	It("returns ErrRootAlreadyExists when a root user is already present", func(ctx SpecContext) {
		existingRootID := uuid.New()
		existingRoot, err := user.NewRoot(existingRootID, time.Now())
		Expect(err).NotTo(HaveOccurred())

		finder.EXPECT().
			FindRoot(mock.Anything).
			Return(existingRoot, nil).
			Once()

		root, err := uc.Execute(ctx)

		Expect(err).To(MatchError(application.ErrRootAlreadyExists))
		Expect(root).To(BeNil())
	})

	It("returns error and aborts when finder fails with unexpected error", func(ctx SpecContext) {
		dbErr := errors.New("db connection failure")

		finder.EXPECT().
			FindRoot(mock.Anything).
			Return(nil, dbErr).
			Once()

		root, err := uc.Execute(ctx)

		Expect(err).To(MatchError(dbErr))
		Expect(root).To(BeNil())
	})

	It("returns error when user saver fails to persist root user", func(ctx SpecContext) {
		expectedRootID := uuid.New()
		saveErr := errors.New("disk full")

		finder.EXPECT().
			FindRoot(mock.Anything).
			Return(nil, domain.ErrNotFound).
			Once()

		idGen.EXPECT().
			NewID().
			Return(expectedRootID).
			Once()

		saver.EXPECT().
			Save(mock.Anything, mock.Anything).
			Return(saveErr).
			Once()

		root, err := uc.Execute(ctx)

		Expect(err).To(MatchError(saveErr))
		Expect(root).To(BeNil())
	})
})
