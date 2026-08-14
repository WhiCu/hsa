package webauthnadapter

import (
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Authenticator Internal", func() {
	It("should stringify loginChallengePayload redacting SessionData", func() {
		payload := loginChallengePayload{
			SessionData: gowebauthn.SessionData{
				Challenge: "super-secret-login-challenge",
			},
		}

		str := payload.String()
		Expect(str).To(ContainSubstring("***REDACTED***"))
		Expect(str).NotTo(ContainSubstring("super-secret-login-challenge"))
	})
})
