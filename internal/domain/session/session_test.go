package session_test

import (
	"net/netip"
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
	ip := netip.MustParseAddr("192.168.1.1")

	Context("Constructor validation", func() {
		It("should redact sensitive fields in String()", func() {
			rt, err := session.New(uuid.New(), uuid.New(), "secret-token-hash", "device", ip, time.Hour, time.Now())
			Expect(err).NotTo(HaveOccurred())

			str := rt.String()
			Expect(str).To(ContainSubstring("***REDACTED***"))
			Expect(str).NotTo(ContainSubstring("secret-token-hash"))

			var nilToken *session.RefreshToken
			Expect(nilToken.String()).To(Equal("<nil>"))
		})

		DescribeTable("Invalid initialization",
			func(id session.RefreshTokenID, uID user.UserID, tokenHash string, ipAddr netip.Addr, expectedErr error) {
				rt, err := session.New(id, uID, tokenHash, "device", ipAddr, time.Hour, time.Now())
				Expect(rt).To(BeNil())
				Expect(err).To(MatchError(domain.ErrValidation))
				Expect(err).To(MatchError(expectedErr))
			},

			Entry("Nil Session ID", uuid.Nil, uuid.New(), "hash", ip, session.ErrIDRequired),
			Entry("Nil User ID", uuid.New(), uuid.Nil, "hash", ip, user.ErrIDRequired),
			Entry("Empty Token Hash", uuid.New(), uuid.New(), "", ip, session.ErrTokenHashRequired),
			Entry("Invalid IP Address", uuid.New(), uuid.New(), "hash", netip.Addr{}, session.ErrIPAddressRequired),
		)
	})

	Context("Getters", func() {
		It("should return correct values", func() {
			id := uuid.New()
			uID := uuid.New()
			now := time.Now()
			ttl := time.Hour

			rt, err := session.New(id, uID, "hash", "device", ip, ttl, now)
			Expect(err).NotTo(HaveOccurred())

			Expect(rt.ID()).To(Equal(id))
			Expect(rt.UserID()).To(Equal(uID))
			Expect(rt.TokenHash()).To(Equal("hash"))
			Expect(rt.DeviceInfo()).To(Equal("device"))
			Expect(rt.IPAddress()).To(Equal(ip))
			Expect(rt.IsRevoked()).To(BeFalse())
			Expect(rt.ExpiresAt()).To(Equal(now.Add(ttl)))
			Expect(rt.CreatedAt()).To(Equal(now))
			Expect(rt.RevokedAt()).To(BeNil())
		})
	})

	Context("Token Lifecycle and Revocation", func() {
		var (
			rt  *session.RefreshToken
			now time.Time
		)

		BeforeEach(func() {
			now = time.Now()
			var err error
			rt, err = session.New(uuid.New(), uuid.New(), "token-hash", "Chrome", ip, time.Hour, now)
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

	Context("Rotate", func() {
		It("should return a new rotated token with updated ID and hash", func() {
			now := time.Now()
			rt, err := session.New(uuid.New(), uuid.New(), "old-hash", "device", ip, time.Hour, now)
			Expect(err).NotTo(HaveOccurred())

			newID := uuid.New()
			rotateNow := now.Add(time.Minute)
			rotated, err := rt.Rotate(newID, "new-hash", rotateNow)
			Expect(err).NotTo(HaveOccurred())

			Expect(rotated.ID()).To(Equal(newID))
			Expect(rotated.TokenHash()).To(Equal("new-hash"))
			Expect(rotated.UserID()).To(Equal(rt.UserID()))
			Expect(rotated.DeviceInfo()).To(Equal(rt.DeviceInfo()))
			Expect(rotated.IPAddress()).To(Equal(rt.IPAddress()))
			Expect(rotated.ExpiresAt()).To(Equal(rt.ExpiresAt()))
			Expect(rotated.CreatedAt()).To(Equal(rotateNow))
			Expect(rotated.RevokedAt()).To(BeNil())
		})

		DescribeTable("Invalid rotation",
			func(newID session.RefreshTokenID, newTokenHash string, expectedErr error) {
				rt, err := session.New(uuid.New(), uuid.New(), "old-hash", "device", ip, time.Hour, time.Now())
				Expect(err).NotTo(HaveOccurred())

				rotated, err := rt.Rotate(newID, newTokenHash, time.Now())
				Expect(rotated).To(BeNil())
				Expect(err).To(MatchError(domain.ErrValidation))
				Expect(err).To(MatchError(expectedErr))
			},
			Entry("Nil Session ID", uuid.Nil, "new-hash", session.ErrIDRequired),
			Entry("Empty Token Hash", uuid.New(), "", session.ErrTokenHashRequired),
		)
	})

	Context("Reconstruct", func() {
		It("should accurately reconstruct a token from storage values", func() {
			id := uuid.New()
			uID := uuid.New()
			hash := "stored-hash"
			device := "mobile"
			now := time.Now()
			expiresAt := now.Add(time.Hour)
			createdAt := now.Add(-time.Hour)
			revokedAt := &now

			rt := session.Reconstruct(id, uID, hash, device, ip, expiresAt, revokedAt, createdAt)

			Expect(rt.ID()).To(Equal(id))
			Expect(rt.UserID()).To(Equal(uID))
			Expect(rt.TokenHash()).To(Equal(hash))
			Expect(rt.DeviceInfo()).To(Equal(device))
			Expect(rt.IPAddress()).To(Equal(ip))
			Expect(rt.ExpiresAt()).To(Equal(expiresAt))
			Expect(rt.CreatedAt()).To(Equal(createdAt))
			Expect(rt.RevokedAt()).To(Equal(revokedAt))
			Expect(rt.IsRevoked()).To(BeTrue())
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
				rt, err := session.New(id, uID, tokenHash, "ua", ip, ttl, now)
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
