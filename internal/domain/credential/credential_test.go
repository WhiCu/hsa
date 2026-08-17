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
	const usbTransport = "usb"

	DescribeTable("Creation and validation checks",
		func(id credential.CredentialID, externalID credential.ExternalID, userID user.UserID, pubKey []byte, expectedErr error) {
			now := time.Now()
			transports := []string{usbTransport, "nfc"}

			c, err := credential.New(id, externalID, userID, pubKey, transports, now)

			if expectedErr != nil {
				Expect(c).To(BeNil())
				Expect(err).To(MatchError(domain.ErrValidation))
				Expect(err).To(MatchError(expectedErr))
			} else {
				Expect(err).NotTo(HaveOccurred())
				Expect(c).NotTo(BeNil())
				Expect(c.ID()).To(Equal(id))
				Expect(c.ExternalID()).To(Equal(externalID))
				Expect(c.UserID()).To(Equal(userID))
				Expect(c.PublicKey()).To(Equal(pubKey))
				Expect(c.SignCount()).To(BeZero())
				Expect(c.Transports()).To(Equal(transports))
				Expect(c.CreatedAt()).To(Equal(now))
				Expect(c.IsRevoked()).To(BeFalse())
				Expect(c.RevokedAt()).To(BeNil())
			}
		},

		Entry("Valid credential", uuid.New(), []byte("external-id"), uuid.New(), []byte("pubkey-bytes"), nil),
		Entry("Nil Credential ID", uuid.Nil, []byte("external-id"), uuid.New(), []byte("pubkey-bytes"), credential.ErrIDRequired),
		Entry("Nil User ID", uuid.New(), []byte("external-id"), uuid.Nil, []byte("pubkey-bytes"), user.ErrIDRequired),
		Entry("Nil External ID", uuid.New(), nil, uuid.New(), []byte("pubkey-bytes"), credential.ErrExternalIDRequired),
		Entry("Empty External ID", uuid.New(), []byte{}, uuid.New(), []byte("pubkey-bytes"), credential.ErrExternalIDRequired),
		Entry("Empty Public Key", uuid.New(), []byte("external-id"), uuid.New(), []byte{}, credential.ErrPublicKeyRequired),
		Entry("Nil Public Key", uuid.New(), []byte("external-id"), uuid.New(), nil, credential.ErrPublicKeyRequired),
	)

	Context("State mutations", func() {
		It("should properly set and update sign count, and detect regression", func() {
			c, err := credential.New(uuid.New(), []byte("external-id"), uuid.New(), []byte("key"), []string{usbTransport}, time.Now())
			Expect(err).NotTo(HaveOccurred())
			Expect(c.SignCount()).To(Equal(uint32(0)))

			// Разрешено устанавливать 0, если текущий signCount == 0
			err = c.SetSignCount(0)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.SignCount()).To(Equal(uint32(0)))

			// Валидное увеличение
			err = c.SetSignCount(10)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.SignCount()).To(Equal(uint32(10)))

			// Проверка регрессии: одинаковое значение
			err = c.SetSignCount(10)
			Expect(err).To(MatchError(credential.ErrSignCountRegression))
			Expect(c.SignCount()).To(Equal(uint32(10)))

			// Проверка регрессии: меньшее значение
			err = c.SetSignCount(5)
			Expect(err).To(MatchError(credential.ErrSignCountRegression))
			Expect(c.SignCount()).To(Equal(uint32(10)))

			// Проверка регрессии: сброс в 0 для ненулевого счётчика
			err = c.SetSignCount(0)
			Expect(err).To(MatchError(credential.ErrSignCountRegression))
			Expect(c.SignCount()).To(Equal(uint32(10)))

			// Валидное дальнейшее увеличение
			err = c.SetSignCount(15)
			Expect(err).NotTo(HaveOccurred())
			Expect(c.SignCount()).To(Equal(uint32(15)))
		})

		It("should properly revoke", func() {
			c, err := credential.New(uuid.New(), []byte("external-id"), uuid.New(), []byte("key"), []string{usbTransport}, time.Now())
			Expect(err).NotTo(HaveOccurred())

			Expect(c.IsRevoked()).To(BeFalse())
			Expect(c.RevokedAt()).To(BeNil())

			now := time.Now()
			err = c.Revoke(now)
			Expect(err).NotTo(HaveOccurred())

			Expect(c.IsRevoked()).To(BeTrue())
			Expect(c.RevokedAt()).NotTo(BeNil())
			Expect(*c.RevokedAt()).To(Equal(now))

			err = c.Revoke(now)
			Expect(err).To(MatchError(credential.ErrAlreadyRevoked))
		})
	})

	Context("Reconstruction", func() {
		It("should properly reconstruct", func() {
			id := uuid.New()
			externalID := []byte("external-id")
			userID := uuid.New()
			pubKey := []byte("pubkey")
			signCount := uint32(42)
			transports := []string{"usb"}
			createdAt := time.Now().Add(-time.Hour)
			revokedAt := time.Now()

			c := credential.Reconstruct(id, externalID, userID, pubKey, signCount, transports, createdAt, &revokedAt)
			Expect(c).NotTo(BeNil())
			Expect(c.ID()).To(Equal(id))
			Expect(c.ExternalID()).To(Equal(externalID))
			Expect(c.UserID()).To(Equal(userID))
			Expect(c.PublicKey()).To(Equal(pubKey))
			Expect(c.SignCount()).To(Equal(signCount))
			Expect(c.Transports()).To(Equal(transports))
			Expect(c.CreatedAt()).To(Equal(createdAt))
			Expect(c.IsRevoked()).To(BeTrue())
			Expect(*c.RevokedAt()).To(Equal(revokedAt))
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
				initialSignCount := rapid.Uint32().Draw(t, "initialSignCount")

				now := time.Now()
				c, err := credential.New(id, externalID, userID, pubKey, transports, now)
				Expect(err).NotTo(HaveOccurred())

				Expect(c.ID()).To(Equal(id))
				Expect(c.ExternalID()).To(Equal(externalID))
				Expect(c.UserID()).To(Equal(userID))
				Expect(c.PublicKey()).To(Equal(pubKey))
				Expect(c.Transports()).To(Equal(transports))
				Expect(c.CreatedAt()).To(Equal(now))

				if initialSignCount > 0 {
					err = c.SetSignCount(initialSignCount)
					Expect(err).NotTo(HaveOccurred())
					Expect(c.SignCount()).To(Equal(initialSignCount))

					// Проверка регрессии в Property-Based тесте
					regressionVal := rapid.Uint32Range(0, initialSignCount).Draw(t, "regressionVal")
					regErr := c.SetSignCount(regressionVal)
					Expect(regErr).To(MatchError(credential.ErrSignCountRegression))
				}
			})
		})
	})

	It("should redact sensitive fields in String()", func() {
		transports := []string{usbTransport, "nfc"}
		c, err := credential.New(uuid.New(), []byte("external-id"), uuid.New(), []byte("public-key"), transports, time.Now())
		Expect(err).NotTo(HaveOccurred())

		str := c.String()
		Expect(str).To(ContainSubstring("***REDACTED***"))
		Expect(str).NotTo(ContainSubstring("external-id"))
		Expect(str).NotTo(ContainSubstring("public-key"))

		var nilCred *credential.Credential
		Expect(nilCred.String()).To(Equal("<nil>"))
	})
})
