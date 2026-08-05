package user_test

import (
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/user"
)

func genValidUUID() *rapid.Generator[uuid.UUID] {
	return rapid.Custom(func(t *rapid.T) uuid.UUID {
		var id uuid.UUID
		for id == uuid.Nil {
			var b [16]byte
			for i := range b {
				b[i] = rapid.Byte().Draw(t, "byte")
			}
			id = uuid.UUID(b)
		}
		return id
	})
}

var _ = Describe("User", func() {
	Context("when creating valid users", func() {
		DescribeTable("Constructor table test",
			func(id, invitedBy uuid.UUID, isRoot bool) {
				now := time.Now()
				var u *user.User
				var err error

				if isRoot {
					u, err = user.NewRoot(id, now)
				} else {
					u, err = user.New(id, invitedBy, now)
				}

				Expect(err).NotTo(HaveOccurred())
				Expect(u).NotTo(BeNil())
				Expect(u.ID()).To(Equal(id))
				Expect(u.IsRoot()).To(Equal(isRoot))
				Expect(u.CreatedAt()).To(Equal(now))

				if isRoot {
					Expect(u.InvitedBy()).To(BeNil())
				} else {
					Expect(u.InvitedBy()).NotTo(BeNil())
					Expect(*u.InvitedBy()).To(Equal(invitedBy))
				}
			},

			Entry("Standard invited user", uuid.New(), uuid.New(), false),
			Entry("Root user", uuid.New(), uuid.Nil, true),
		)
	})

	Context("when creating invalid users", func() {
		DescribeTable("Validation error checking",
			func(id, invitedBy uuid.UUID, expectedSubErr error) {
				u, err := user.New(id, invitedBy, time.Now())
				Expect(u).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(domain.ErrValidation))
				Expect(err).To(MatchError(expectedSubErr))
			},

			Entry("Nil user ID", uuid.Nil, uuid.New(), user.ErrIDRequired),
			Entry("Nil invitedBy ID", uuid.New(), uuid.Nil, user.ErrInvitedBy),
			Entry("Both Nil IDs", uuid.Nil, uuid.Nil, user.ErrIDRequired),
		)

		It("should fail NewRoot if ID is Nil", func() {
			u, err := user.NewRoot(uuid.Nil, time.Now())
			Expect(u).To(BeNil())
			Expect(err).To(MatchError(domain.ErrValidation))
			Expect(err).To(MatchError(user.ErrIDRequired))
		})
	})

	Context("User Property-Based Testing", func() {
		It("should create valid standard users with arbitrary non-nil UUIDs", func() {
			rapid.Check(GinkgoT(), func(t *rapid.T) {
				id := genValidUUID().Draw(t, "user_id")
				invitedBy := genValidUUID().Draw(t, "invited_by_id")
				createdAt := time.Unix(rapid.Int64().Draw(t, "timestamp"), 0)

				u, err := user.New(id, invitedBy, createdAt)
				Expect(err).NotTo(HaveOccurred())
				Expect(u.ID()).To(Equal(id))
				Expect(u.IsRoot()).To(BeFalse())
				Expect(*u.InvitedBy()).To(Equal(invitedBy))
				Expect(u.CreatedAt()).To(Equal(createdAt))
			})
		})

		It("should create valid root users with arbitrary non-nil UUIDs", func() {
			rapid.Check(GinkgoT(), func(t *rapid.T) {
				id := genValidUUID().Draw(t, "root_user_id")
				createdAt := time.Unix(rapid.Int64().Draw(t, "timestamp"), 0)

				u, err := user.NewRoot(id, createdAt)
				Expect(err).NotTo(HaveOccurred())
				Expect(u.ID()).To(Equal(id))
				Expect(u.IsRoot()).To(BeTrue())
				Expect(u.InvitedBy()).To(BeNil())
			})
		})
	})
	Context("User Helper Functions", func() {
		It("Create UserID", func() {
			rapid.Check(GinkgoT(), func(t *rapid.T) {
				bytes := rapid.SliceOfN(rapid.Byte(), 16, 16).Draw(t, "rawUUID")
				id, err := user.NewUserID(bytes)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).To(Equal(uuid.UUID(bytes)))
			})
		})
	})
})
