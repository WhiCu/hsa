package webauthnadapter_test

import (
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/knadh/koanf/v2"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/do/v2"

	webauthnadapter "github.com/whicu/hsa/internal/infrastructure/auth/webauthn"
	"github.com/whicu/hsa/internal/infrastructure/auth/webauthn/mocks"
	"github.com/whicu/hsa/pkg/logger"
)

const testAppConst = "Test App"
const testDisplayNameConst = "Test"
const localhostConst = "localhost"
var _ = Describe("Package DI", func() {
	var injector do.Injector

	BeforeEach(func() {
		injector = do.New(webauthnadapter.Package)
	})

	It("fails to resolve config when koanf is not provided", func() {
		_, err := do.Invoke[webauthnadapter.Config](injector)
		Expect(err).To(HaveOccurred())
	})

	It("fails to resolve webauthn when config is missing", func() {
		_, err := do.Invoke[*webauthn.WebAuthn](injector)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invoke config"))
	})

	It("fails to resolve webauthn when config is invalid", func() {
		do.OverrideValue(injector, webauthnadapter.Config{
			RP: webauthnadapter.RPConfig{
				ID:          "",
				DisplayName: testAppConst,
			},
		})
		_, err := do.Invoke[*webauthn.WebAuthn](injector)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("init go-webauthn"))
	})

	It("fails to resolve authenticator when dependencies are missing", func() {
		_, err := do.Invoke[*webauthnadapter.Authenticator](injector)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invoke logger"))

		do.OverrideValue(injector, logger.NewNOPSlog())
		_, err = do.Invoke[*webauthnadapter.Authenticator](injector)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invoke webauthn"))

		do.OverrideValue(injector, webauthnadapter.Config{
			RP: webauthnadapter.RPConfig{ID: ""},
		})
		_, err = do.Invoke[*webauthnadapter.Authenticator](injector)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invoke webauthn"))

		wa, _ := webauthn.New(&webauthn.Config{RPID: localhostConst, RPDisplayName: testAppConst})
		do.OverrideValue(injector, wa)
		_, err = do.Invoke[*webauthnadapter.Authenticator](injector)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invoke challenge codec"))

		codec := mocks.NewChallengeCodec(GinkgoT())
		do.OverrideValue[webauthnadapter.ChallengeCodec](injector, codec)
		_, err = do.Invoke[*webauthnadapter.Authenticator](injector)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invoke credentials provider"))
	})

	It("fails to resolve registrator when dependencies are missing", func() {
		_, err := do.Invoke[*webauthnadapter.Registrator](injector)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invoke logger"))

		do.OverrideValue(injector, logger.NewNOPSlog())
		_, err = do.Invoke[*webauthnadapter.Registrator](injector)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invoke webauthn"))

		wa, _ := webauthn.New(&webauthn.Config{RPID: localhostConst, RPDisplayName: testAppConst})
		do.OverrideValue(injector, wa)
		_, err = do.Invoke[*webauthnadapter.Registrator](injector)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invoke challenge codec"))
	})

	It("resolves config to default if invalid koanf provided but fails validation", func() {
		k := koanf.New(".")
		do.OverrideValue(injector, k)

		_, err := do.Invoke[webauthnadapter.Config](injector)
		Expect(err).To(HaveOccurred())
	})

	It("fails to resolve authenticator when config fails in authenticator invoke", func() {
		wa, _ := webauthn.New(&webauthn.Config{RPID: localhostConst, RPDisplayName: testAppConst})
		do.OverrideValue(injector, wa)
		codec := mocks.NewChallengeCodec(GinkgoT())
		do.OverrideValue[webauthnadapter.ChallengeCodec](injector, codec)
		credsProv := mocks.NewCredentialsProvider(GinkgoT())
		do.OverrideValue[webauthnadapter.CredentialsProvider](injector, credsProv)
		do.OverrideValue(injector, logger.NewNOPSlog())

		k := koanf.New(".")
		do.OverrideValue(injector, k)

		_, err := do.Invoke[*webauthnadapter.Authenticator](injector)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invoke config"))
	})

	It("fails to resolve registrator when config fails in registrator invoke", func() {
		wa, _ := webauthn.New(&webauthn.Config{RPID: localhostConst, RPDisplayName: testAppConst})
		do.OverrideValue(injector, wa)
		codec := mocks.NewChallengeCodec(GinkgoT())
		do.OverrideValue[webauthnadapter.ChallengeCodec](injector, codec)
		do.OverrideValue(injector, logger.NewNOPSlog())

		k := koanf.New(".")
		do.OverrideValue(injector, k)

		_, err := do.Invoke[*webauthnadapter.Registrator](injector)
		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("invoke config"))
	})
})
