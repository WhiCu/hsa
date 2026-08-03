package credential_test

import (
	"time"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"pgregory.net/rapid"

	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/credential"
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

var _ = Describe("Credential", func() {

	DescribeTable("Creation and validation checks",
		func(id credential.CredentialID, externalID credential.ExternalID, userID user.UserID, pubKey []byte, expectedErr error) {
			now := time.Now()
			transports := []string{"usb", "nfc"}

			c, err := credential.New(id, externalID, userID, pubKey, transports, now)

			if expectedErr != nil {
				Expect(c).To(BeNil())
				Expect(err).To(MatchError(domain.ErrValidation))
				Expect(err).To(MatchError(expectedErr))
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.ID()).To(Equal(id))
				Expect(c.UserID()).To(Equal(userID))
				Expect(c.SignCount()).To(BeZero())
			}
		},

		Entry("Valid credential", uuid.New(), []byte("external-id"), uuid.New(), []byte("pubkey-bytes"), nil),
		Entry("Nil Credential ID", uuid.Nil, []byte("external-id"), uuid.New(), []byte("pubkey-bytes"), credential.ErrIDRequired),
		Entry("Nil User ID", uuid.New(), []byte("external-id"), uuid.Nil, []byte("pubkey-bytes"), user.ErrIDRequired),
		Entry("Nil External ID", uuid.New(), nil, uuid.New(), []byte("pubkey-bytes"), credential.ErrExternalIDRequired),
		Entry("Empty Public Key", uuid.New(), []byte("external-id"), uuid.New(), []byte{}, credential.ErrPublicKeyRequired),
		Entry("Nil Public Key", uuid.New(), []byte("external-id"), uuid.New(), nil, credential.ErrPublicKeyRequired),
	)

	Context("State mutations", func() {
		It("should properly set and update sign count", func() {
			c, err := credential.New(uuid.New(), []byte("external-id"), uuid.New(), []byte("key"), nil, time.Now())
			Expect(err).NotTo(HaveOccurred())
			Expect(c.SignCount()).To(Equal(uint32(0)))

			c.SetSignCount(42)
			Expect(c.SignCount()).To(Equal(uint32(42)))
		})
	})

	Context("Property-Based Testing", func() {
		It("should correctly instantiate valid credentials for arbitrary properties", func() {
			rapid.Check(GinkgoT(), func(t *rapid.T) {
				id := genValidUUID().Draw(t, "id")
				externalID := rapid.SliceOfN(rapid.Byte(), 1, 256).Draw(t, "externalID")
				userID := genValidUUID().Draw(t, "userID")
				pubKey := rapid.SliceOfN(rapid.Byte(), 1, 256).Draw(t, "pubKey")
				transports := rapid.SliceOf(rapid.String()).Draw(t, "transports")
				signCount := rapid.Uint32().Draw(t, "signCount")

				c, err := credential.New(id, externalID, userID, pubKey, transports, time.Now())
				Expect(err).NotTo(HaveOccurred())

				c.SetSignCount(signCount)
				Expect(c.SignCount()).To(Equal(signCount))
				Expect(c.ID()).To(Equal(id))
				Expect(c.UserID()).To(Equal(userID))
			})
		})
	})
})
