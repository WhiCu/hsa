package key_test

import (
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/credential"
	"github.com/whicu/hsa/internal/domain/key"
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

var _ = Describe("WrappedKey", func() {

	Context("Scope validation", func() {
		DescribeTable("Valid and invalid scopes",
			func(s key.Scope, expectedValid bool) {
				Expect(s.Valid()).To(Equal(expectedValid))
			},
			Entry("Main Scope", key.ScopeMain, true),
			Entry("Decoy Scope", key.ScopeDecoy, true),
			Entry("Invalid Scope Low", key.Scope(99), false),
		)
	})

	Context("Policy checks", func() {
		It("should check wrapped key count limits strictly", func() {
			policy := key.NewPolicy(5)

			Expect(policy.ValidateCount(0)).To(Succeed())
			Expect(policy.ValidateCount(4)).To(Succeed())
			Expect(policy.ValidateCount(5)).To(Succeed())
			Expect(policy.ValidateCount(6)).To(MatchError(key.ErrTooManyWrappedKeys))
		})
	})

	Context("Constructor validation", func() {
		DescribeTable("Invalid creation parameters",
			func(id key.WrappedKeyID, uID user.UserID, scope key.Scope, dek []byte, alg string, expectedErr error) {
				cID := uuid.New()
				wk, err := key.New(id, uID, &cID, scope, dek, alg, time.Now())
				Expect(wk).To(BeNil())
				Expect(err).To(MatchError(domain.ErrValidation))
				Expect(err).To(MatchError(expectedErr))
			},

			Entry("Nil Key ID", uuid.Nil, uuid.New(), key.ScopeMain, []byte("dek"), "AES-256-GCM", key.ErrIDRequired),
			Entry("Nil User ID", uuid.New(), uuid.Nil, key.ScopeMain, []byte("dek"), "AES-256-GCM", user.ErrIDRequired),
			Entry("Invalid Scope", uuid.New(), uuid.New(), key.Scope(255), []byte("dek"), "AES-256-GCM", key.ErrScopeInvalid),
			Entry("Empty Wrapped DEK", uuid.New(), uuid.New(), key.ScopeMain, []byte{}, "AES-256-GCM", key.ErrWrappedDEKRequired),
			Entry("Empty Wrap Alg", uuid.New(), uuid.New(), key.ScopeMain, []byte("dek"), "", key.ErrWrapAlgorithmRequired),
		)
	})

	Context("Property-Based Testing", func() {
		It("should redact sensitive fields in String()", func() {
			cID := uuid.New()
			wk, err := key.New(uuid.New(), uuid.New(), &cID, key.ScopeMain, []byte("super-secret-dek"), "AES-256-GCM", time.Now())
			Expect(err).NotTo(HaveOccurred())

			str := wk.String()
			Expect(str).To(ContainSubstring("***REDACTED***"))
			Expect(str).NotTo(ContainSubstring("super-secret-dek"))

			var nilKey *key.WrappedKey
			Expect(nilKey.String()).To(Equal("<nil>"))
		})

		It("should handle nil credentialID in String()", func() {
			wk, err := key.New(uuid.New(), uuid.New(), nil, key.ScopeMain, []byte("super-secret-dek"), "AES-256-GCM", time.Now())
			Expect(err).NotTo(HaveOccurred())

			str := wk.String()
			Expect(str).To(ContainSubstring("credentialID: <nil>"))
		})

		It("should construct valid keys for any valid parameter combination", func() {
			rapid.Check(GinkgoT(), func(t *rapid.T) {
				id := genValidUUID().Draw(t, "id")
				userID := genValidUUID().Draw(t, "userID")

				var credID *credential.CredentialID
				if rapid.Bool().Draw(t, "hasCredID") {
					cid := genValidUUID().Draw(t, "credID")
					credID = &cid
				}

				scope := rapid.SampledFrom([]key.Scope{key.ScopeMain, key.ScopeDecoy}).Draw(t, "scope")
				dek := rapid.SliceOfN(rapid.Byte(), 16, 64).Draw(t, "dek")
				alg := rapid.StringMatching(`[A-Z0-9\-]{3,10}`).Draw(t, "alg")

				wk, err := key.New(id, userID, credID, scope, dek, alg, time.Now())
				Expect(err).NotTo(HaveOccurred())
				Expect(wk.ID()).To(Equal(id))
				Expect(wk.Scope()).To(Equal(scope))
				Expect(wk.CredentialID()).To(Equal(credID))
			})
		})

		It("should evaluate policy limits deterministically", func() {
			rapid.Check(GinkgoT(), func(t *rapid.T) {
				maxKeys := rapid.IntRange(1, 100).Draw(t, "maxKeys")
				count := rapid.IntRange(0, 200).Draw(t, "count")

				policy := key.NewPolicy(maxKeys)
				err := policy.ValidateCount(count)

				if count > maxKeys {
					Expect(err).To(MatchError(key.ErrTooManyWrappedKeys))
				} else {
					Expect(err).NotTo(HaveOccurred())
				}
			})
		})
	})
})
