package key_test

import (
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/whicu/hsa/internal/domain/key"
)

var _ = Describe("WrappedKey Coverage", func() {
	Describe("Scope String conversions", func() {
		It("should convert Scope to string", func() {
			Expect(key.ScopeToString(key.ScopeMain)).To(Equal("main"))
			Expect(key.ScopeToString(key.ScopeDecoy)).To(Equal("decoy"))
		})

		It("should convert string to Scope", func() {
			Expect(key.ScopeFromString("main")).To(Equal(key.ScopeMain))
			Expect(key.ScopeFromString("decoy")).To(Equal(key.ScopeDecoy))
		})
	})

	Describe("WrappedKey Getter and Constructor Methods", func() {
		It("should correctly return all fields from New", func() {
			id := uuid.New()
			uID := uuid.New()
			cID := uuid.New()
			scope := key.ScopeMain
			dek := []byte("secret")
			alg := "AES-256-GCM"
			now := time.Now()

			wk, err := key.New(id, uID, cID, scope, dek, alg, now)
			Expect(err).NotTo(HaveOccurred())

			Expect(wk.ID()).To(Equal(id))
			Expect(wk.UserID()).To(Equal(uID))
			Expect(wk.Scope()).To(Equal(scope))
			Expect(wk.CredentialID()).To(Equal(cID))
			Expect(wk.WrappedDEK()).To(Equal(dek))
			Expect(wk.WrapAlgorithm()).To(Equal(alg))
			Expect(wk.CreatedAt()).To(Equal(now))
		})

		It("should correctly construct and return fields via Reconstruct", func() {
			id := uuid.New()
			uID := uuid.New()
			cID := uuid.New()
			scope := key.ScopeDecoy
			dek := []byte("another-secret")
			alg := "ChaCha20-Poly1305"
			now := time.Now()

			wk := key.Reconstruct(id, uID, cID, scope, dek, alg, now)

			Expect(wk.ID()).To(Equal(id))
			Expect(wk.UserID()).To(Equal(uID))
			Expect(wk.Scope()).To(Equal(scope))
			Expect(wk.CredentialID()).To(Equal(cID))
			Expect(wk.WrappedDEK()).To(Equal(dek))
			Expect(wk.WrapAlgorithm()).To(Equal(alg))
			Expect(wk.CreatedAt()).To(Equal(now))
		})
	})
})
