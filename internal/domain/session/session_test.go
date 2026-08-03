package session_test

import (
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/session"
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

var _ = Describe("RefreshToken Domain", func() {

	Context("Constructor validation", func() {
		DescribeTable("Invalid initialization",
			func(id session.RefreshTokenID, uID user.UserID, tokenHash string, expectedErr error) {
				rt, err := session.New(id, uID, tokenHash, "device", "127.0.0.1", time.Hour, time.Now())
				Expect(rt).To(BeNil())
				Expect(err).To(MatchError(domain.ErrValidation))
				Expect(err).To(MatchError(expectedErr))
			},

			Entry("Nil Session ID", uuid.Nil, uuid.New(), "hash", session.ErrIDRequired),
			Entry("Nil User ID", uuid.New(), uuid.Nil, "hash", user.ErrIDRequired),
			Entry("Empty Token Hash", uuid.New(), uuid.New(), "", session.ErrTokenHashRequired),
		)
	})

	Context("Token Lifecycle and Revocation", func() {
		var (
			rt  *session.RefreshToken
			now time.Time
		)

		BeforeEach(func() {
			now = time.Now()
			var err error
			rt, err = session.New(uuid.New(), uuid.New(), "token-hash", "Chrome", "192.168.1.1", time.Hour, now)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should be valid before expiration and when not revoked", func() {
			Expect(rt.IsValid(now.Add(30 * time.Minute))).To(BeTrue())
		})

		It("should be invalid after expiration", func() {
			Expect(rt.IsValid(now.Add(61 * time.Minute))).To(BeFalse())
		})

		It("should become invalid once revoked", func() {
			revokeTime := now.Add(10 * time.Minute)
			err := rt.Revoke(revokeTime)
			Expect(err).NotTo(HaveOccurred())

			Expect(rt.IsValid(now.Add(15 * time.Minute))).To(BeFalse())
		})

		It("should fail to revoke more than once", func() {
			revokeTime := now.Add(10 * time.Minute)
			Expect(rt.Revoke(revokeTime)).To(Succeed())
			Expect(rt.Revoke(revokeTime)).To(MatchError(session.ErrAlreadyRevoked))
		})
	})

	Context("Property-Based Testing", func() {
		It("should validate token validity state accurately based on time and revocation state", func() {
			rapid.Check(GinkgoT(), func(t *rapid.T) {
				id := genValidUUID().Draw(t, "id")
				uID := genValidUUID().Draw(t, "userID")
				tokenHash := rapid.StringMatching(`[a-f0-9]{64}`).Draw(t, "hash")
				ttlSec := rapid.Int64Range(10, 3600).Draw(t, "ttl")
				ttl := time.Duration(ttlSec) * time.Second

				now := time.Unix(1700000000, 0)
				rt, err := session.New(id, uID, tokenHash, "ua", "127.0.0.1", ttl, now)
				Expect(err).NotTo(HaveOccurred())

				checkTimeSec := rapid.Int64Range(0, 7200).Draw(t, "checkTimeOffset")
				checkTime := now.Add(time.Duration(checkTimeSec) * time.Second)

				shouldRevoke := rapid.Bool().Draw(t, "shouldRevoke")
				if shouldRevoke {
					Expect(rt.Revoke(now)).To(Succeed())
				}

				isValid := rt.IsValid(checkTime)

				if shouldRevoke {
					Expect(isValid).To(BeFalse(), "Revoked token must be invalid")
				} else if checkTime.Before(now.Add(ttl)) {
					Expect(isValid).To(BeTrue(), "Unrevoked token before TTL must be valid")
				} else {
					Expect(isValid).To(BeFalse(), "Token after TTL must be invalid")
				}
			})
		})
	})
})
