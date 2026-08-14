package application_test

import (
	"net/netip"
	"strings"
	"testing"

	"github.com/whicu/hsa/internal/application"
)

func TestFinishInviteRegistrationInput_String(t *testing.T) {
	t.Parallel()

	in := application.FinishInviteRegistrationInput{
		ChallengeToken:       "secret_challenge",
		RegistrationResponse: []byte("secret_response"),
		DeviceInfo:           "Test Device",
		IPAddress:            netip.MustParseAddr("192.168.1.100"),
	}
	str := in.String()
	if strings.Contains(str, "secret_challenge") {
		t.Errorf("ChallengeToken was not redacted")
	}
	if strings.Contains(str, "secret_response") {
		t.Errorf("RegistrationResponse was not redacted")
	}
	if !strings.Contains(str, "Test Device") {
		t.Errorf("DeviceInfo was omitted")
	}
}

func TestFinishInviteRegistrationOutput_String(t *testing.T) {
	t.Parallel()

	out := application.FinishInviteRegistrationOutput{
		AccessToken:  "secret_access",
		RefreshToken: "secret_refresh",
	}
	str := out.String()
	if strings.Contains(str, "secret_access") {
		t.Errorf("AccessToken was not redacted")
	}
	if strings.Contains(str, "secret_refresh") {
		t.Errorf("RefreshToken was not redacted")
	}
}

func TestRegistrationResult_String(t *testing.T) {
	res := application.RegistrationResult{
		PublicKey: []byte("secret_public_key"),
	}
	str := res.String()
	if strings.Contains(str, "secret_public_key") {
		t.Errorf("PublicKey was not redacted")
	}
}

func TestAuthenticationResult_String(t *testing.T) {
	res := application.AuthenticationResult{
		ExternalID: []byte("secret_external_id"),
	}
	str := res.String()
	if strings.Contains(str, "secret_external_id") {
		t.Errorf("ExternalID was not redacted")
	}
}

func TestLoginInput_String(t *testing.T) {
	t.Parallel()

	in := application.LoginInput{
		ChallengeToken:         "secret_challenge",
		AuthenticationResponse: []byte("secret_auth"),
		DeviceInfo:             "Test Device",
		IPAddress:              netip.MustParseAddr("192.168.1.100"),
	}
	str := in.String()
	if strings.Contains(str, "secret_challenge") {
		t.Errorf("ChallengeToken was not redacted")
	}
	if strings.Contains(str, "secret_auth") {
		t.Errorf("AuthenticationResponse was not redacted")
	}
	if !strings.Contains(str, "Test Device") {
		t.Errorf("DeviceInfo was omitted")
	}
}

func TestLoginOutput_String(t *testing.T) {
	t.Parallel()

	out := application.LoginOutput{
		AccessToken:  "secret_access",
		RefreshToken: "secret_refresh",
	}
	str := out.String()
	if strings.Contains(str, "secret_access") {
		t.Errorf("AccessToken was not redacted")
	}
	if strings.Contains(str, "secret_refresh") {
		t.Errorf("RefreshToken was not redacted")
	}
}
