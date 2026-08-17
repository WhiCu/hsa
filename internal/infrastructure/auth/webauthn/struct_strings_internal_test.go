package webauthnadapter

import (
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

)

var _ = Describe("webauthnUser String()", func() {
	It("should redact credentials array", func() {
		u := &webauthnUser{
			id: uuid.New(),
			credentials: []gowebauthn.Credential{
				{
					ID:        []byte("secret-credential-id"),
					PublicKey: []byte("secret-public-key"),
				},
			},
		}

		str := u.String()
		Expect(str).NotTo(ContainSubstring("secret-credential-id"), "webauthnUser String() leaked credentials ID")
		Expect(str).NotTo(ContainSubstring("secret-public-key"), "webauthnUser String() leaked credentials PublicKey")
		Expect(str).To(ContainSubstring("***REDACTED***"), "webauthnUser String() did not contain ***REDACTED***")
	})

	It("should handle nil pointer gracefully", func() {
		var nilU *webauthnUser
		Expect(nilU.String()).To(Equal("<nil>"))
	})
})
