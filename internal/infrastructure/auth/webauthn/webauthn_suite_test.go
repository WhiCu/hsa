package webauthnadapter_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestWebauthn(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Webauthn Suite")
}
