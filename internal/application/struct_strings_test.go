package application_test

import (
	"strings"
	"testing"

	"github.com/whicu/hsa/internal/application"
)

func TestFinishInviteRegistrationInput_String(t *testing.T) {
	in := application.FinishInviteRegistrationInput{
		ChallengeToken:       "secret_challenge",
		RegistrationResponse: []byte("secret_response"),
		DeviceInfo:           "Test Device",
		IPAddress:            "192.168.1.50",
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

func TestLoginInput_String(t *testing.T) {
	in := application.LoginInput{
		ChallengeToken:         "secret_challenge",
		AuthenticationResponse: []byte("secret_auth"),
		DeviceInfo:             "Test Device",
		IPAddress:              "192.168.1.50",
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
