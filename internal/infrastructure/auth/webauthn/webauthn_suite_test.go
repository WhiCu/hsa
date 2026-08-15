package webauthnadapter_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	webauthnadapter "github.com/whicu/hsa/internal/infrastructure/auth/webauthn"
)

func TestWebauthn(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Webauthn Suite")
}

const testChallengeToken = "test-challenge-token"

var testConfig = webauthnadapter.Config{
	RP: webauthnadapter.RPConfig{
		ID:          "localhost",
		DisplayName: "Test App",
		Origins:     []string{"http://localhost:8080"},
	},
	ChallengeTTL: time.Minute,
}
