package webauthnadapter

import (
	"errors"
	"time"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/whicu/hsa/internal/domain/credential"
	"github.com/whicu/hsa/internal/domain/user"
	"github.com/whicu/hsa/internal/infrastructure/auth/webauthn/mocks"
	"github.com/whicu/hsa/pkg/logger"
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

	It("toWebAuthnCredential correctly maps fields", func() {
		pubKey := []byte("pub")
		cred, err := credential.New(
			uuid.New(), []byte("ext"), uuid.New(), pubKey, []string{"usb"}, time.Now(),
		)
		Expect(err).NotTo(HaveOccurred())
		err = cred.SetSignCount(10)
		Expect(err).NotTo(HaveOccurred())

		waCred := toWebAuthnCredential(cred)
		Expect(waCred.ID).To(Equal([]byte("ext")))
		Expect(waCred.PublicKey).To(Equal(pubKey))
		Expect(waCred.Authenticator.SignCount).To(Equal(uint32(10)))
	})

	It("tests internal validate func mapping behavior and error conditions", func(ctx SpecContext) {
		wa := newTestWebAuthn(GinkgoT())
		codec := mocks.NewChallengeCodec(GinkgoT())
		credsProv := mocks.NewCredentialsProvider(GinkgoT())
		auth := NewAuthenticator(logger.NewNOPSlog(), wa, codec, credsProv, time.Minute)

		var userIDOut user.UserID
		f := auth.getValidateFunc(ctx, &userIDOut)

		_, err := f(nil, []byte("invalid-uuid"))
		Expect(err).To(MatchError(ErrUserHandleInvalid))

		validID := uuid.New()

		credsProv.EXPECT().FindByUserID(ctx, validID).Return(nil, errors.New("db error")).Once()
		_, err = f(nil, validID[:])
		Expect(err).To(MatchError("db error"))

		pubKey := []byte("pub")
		cred, _ := credential.New(
			validID, []byte("ext"), validID, pubKey, []string{"usb"}, time.Now(),
		)
		credsProv.EXPECT().FindByUserID(ctx, validID).Return([]*credential.Credential{cred}, nil).Once()

		waUser, err := f(nil, validID[:])
		Expect(err).NotTo(HaveOccurred())
		Expect(waUser.WebAuthnName()).To(Equal(validID.String()))
		Expect(waUser.WebAuthnCredentials()).To(HaveLen(1))
	})
})
