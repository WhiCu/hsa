package invite_test

import (
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/invite"
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

var _ = Describe("Invite", func() {

	Context("Invite Creation", func() {
		DescribeTable("Validation failures",
			func(id invite.InviteID, createdBy user.UserID, codeHash string, expectedErr error) {
				inv, err := invite.New(id, createdBy, codeHash, time.Hour, time.Now())
				Expect(inv).To(BeNil())
				Expect(err).To(MatchError(domain.ErrValidation))
				Expect(err).To(MatchError(expectedErr))
			},

			Entry("Nil Invite ID", uuid.Nil, uuid.New(), "hash123", invite.ErrIDRequired),
			Entry("Nil CreatedBy ID", uuid.New(), uuid.Nil, "hash123", user.ErrIDRequired),
			Entry("Empty Code Hash", uuid.New(), uuid.New(), "", invite.ErrCodeHashRequired),
		)
	})

	Context("Redeem lifecycle", func() {
		var (
			inv       *invite.Invite
			now       time.Time
			createdBy user.UserID
			redeemer  user.UserID
		)

		BeforeEach(func() {
			now = time.Now()
			createdBy = uuid.New()
			redeemer = uuid.New()
			var err error
			inv, err = invite.New(uuid.New(), createdBy, "valid-hash", time.Hour, now)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should be successfully redeemed when active and not expired", func() {
			Expect(inv.IsUsed()).To(BeFalse())
			Expect(inv.IsExpired(now)).To(BeFalse())

			err := inv.Redeem(redeemer, now.Add(30*time.Minute))
			Expect(err).NotTo(HaveOccurred())
			Expect(inv.IsUsed()).To(BeTrue())
		})

		It("should fail redeeming if expired", func() {
			futureTime := now.Add(2 * time.Hour)
			Expect(inv.IsExpired(futureTime)).To(BeTrue())

			err := inv.Redeem(redeemer, futureTime)
			Expect(err).To(MatchError(invite.ErrExpired))
		})

		It("should fail redeeming twice (already used)", func() {
			err := inv.Redeem(redeemer, now.Add(10*time.Minute))
			Expect(err).NotTo(HaveOccurred())

			err2 := inv.Redeem(redeemer, now.Add(20*time.Minute))
			Expect(err2).To(MatchError(invite.ErrAlreadyUsed))
		})
	})

	Context("Policy checks", func() {
		It("should check active count limits strictly", func() {
			policy := invite.NewPolicy(5)

			Expect(policy.CanIssue(0)).To(Succeed())
			Expect(policy.CanIssue(4)).To(Succeed())
			Expect(policy.CanIssue(5)).To(MatchError(invite.ErrTooManyActive))
			Expect(policy.CanIssue(6)).To(MatchError(invite.ErrTooManyActive))
		})
	})

	Context("Property-Based Testing", func() {
		It("should respect expiration boundaries for arbitrary TTLs", func() {
			rapid.Check(GinkgoT(), func(t *rapid.T) {
				id := genValidUUID().Draw(t, "id")
				createdBy := genValidUUID().Draw(t, "createdBy")
				codeHash := rapid.StringMatching(`[a-f0-9]{32}`).Draw(t, "hash")

				ttlSeconds := rapid.Int64Range(1, 10000).Draw(t, "ttl")
				ttl := time.Duration(ttlSeconds) * time.Second

				now := time.Unix(1700000000, 0)
				inv, err := invite.New(id, createdBy, codeHash, ttl, now)
				Expect(err).NotTo(HaveOccurred())

				beforeExpiry := now.Add(ttl - time.Nanosecond)
				afterExpiry := now.Add(ttl + time.Nanosecond)

				Expect(inv.IsExpired(beforeExpiry)).To(BeFalse())
				Expect(inv.IsExpired(afterExpiry)).To(BeTrue())
			})
		})

		It("should evaluate policy limits deterministically", func() {
			rapid.Check(GinkgoT(), func(t *rapid.T) {
				maxActive := rapid.IntRange(1, 100).Draw(t, "maxActive")
				activeCount := rapid.IntRange(0, 200).Draw(t, "activeCount")

				policy := invite.NewPolicy(maxActive)
				err := policy.CanIssue(activeCount)

				if activeCount >= maxActive {
					Expect(err).To(MatchError(invite.ErrTooManyActive))
				} else {
					Expect(err).NotTo(HaveOccurred())
				}
			})
		})
	})
})
