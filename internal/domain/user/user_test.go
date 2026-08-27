package user_test

import (
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

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
			func(id uuid.UUID, role user.Role, invitedBy uuid.UUID, isRoot bool) {
				now := time.Now()
				var u *user.User
				var err error

				if isRoot {
					u, err = user.NewRoot(id, now)
				} else {
					u, err = user.New(id, role, invitedBy, now)
				}

				Expect(err).NotTo(HaveOccurred())
				Expect(u).NotTo(BeNil())
				Expect(u.ID()).To(Equal(id))
				Expect(u.IsRoot()).To(Equal(isRoot))
				Expect(u.CreatedAt()).To(Equal(now))

				if isRoot {
					Expect(u.Role()).To(Equal(user.Admin))
					Expect(u.InvitedBy()).To(BeNil())
				} else {
					Expect(u.Role()).To(Equal(role))
					Expect(u.InvitedBy()).NotTo(BeNil())
					Expect(*u.InvitedBy()).To(Equal(invitedBy))
				}
			},

			Entry("Standard invited user (Member)", uuid.New(), user.Member, uuid.New(), false),
			Entry("Standard invited user (Admin)", uuid.New(), user.Admin, uuid.New(), false),
			Entry("Root user", uuid.New(), user.Admin, uuid.Nil, true),
		)
	})

	Context("when creating invalid users", func() {
		DescribeTable("Validation error checking",
			func(id uuid.UUID, role user.Role, invitedBy uuid.UUID, expectedSubErr error) {
				u, err := user.New(id, role, invitedBy, time.Now())
				Expect(u).To(BeNil())
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(expectedSubErr))
			},

			Entry("Nil user ID", uuid.Nil, user.Member, uuid.New(), user.ErrIDRequired),
			Entry("Unknown role", uuid.New(), user.Unknown, uuid.New(), user.ErrRoleRequired),
			Entry("Nil invitedBy ID", uuid.New(), user.Member, uuid.Nil, user.ErrInvitedBy),
			Entry("Multiple errors (ID checked first)", uuid.Nil, user.Unknown, uuid.Nil, user.ErrIDRequired),
		)

		It("should fail NewRoot if ID is Nil", func() {
			u, err := user.NewRoot(uuid.Nil, time.Now())
			Expect(u).To(BeNil())
			Expect(err).To(MatchError(user.ErrIDRequired))
		})
	})

	Context("Role Property and Behavior", func() {
		DescribeTable("RoleFromString",
			func(slug string, expectedRole user.Role, expectedErr error) {
				role, err := user.RoleFromString(slug)
				if expectedErr != nil {
					Expect(err).To(MatchError(expectedErr))
				} else {
					Expect(err).NotTo(HaveOccurred())
					Expect(role).To(Equal(expectedRole))
				}
			},
			Entry("Member", "member", user.Member, nil),
			Entry("Admin", "admin", user.Admin, nil),
			Entry("Unknown string", "superadmin", user.Unknown, user.ErrUnknownRole),
			Entry("Empty string", "", user.Unknown, user.ErrUnknownRole),
		)

		It("should return correct string representation", func() {
			Expect(user.Member.String()).To(Equal("member"))
			Expect(user.Admin.String()).To(Equal("admin"))
			Expect(user.Unknown.String()).To(Equal(""))
		})
	})

	Context("User Property-Based Testing", func() {
		It("should create valid standard users with arbitrary non-nil UUIDs", func() {
			rapid.Check(GinkgoT(), func(t *rapid.T) {
				id := genValidUUID().Draw(t, "user_id")
				invitedBy := genValidUUID().Draw(t, "invited_by_id")
				createdAt := time.Unix(rapid.Int64().Draw(t, "timestamp"), 0)

				// Для property-base тестирования используем валидную роль
				u, err := user.New(id, user.Member, invitedBy, createdAt)
				Expect(err).NotTo(HaveOccurred())
				Expect(u.ID()).To(Equal(id))
				Expect(u.Role()).To(Equal(user.Member))
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
				Expect(u.Role()).To(Equal(user.Admin))
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
