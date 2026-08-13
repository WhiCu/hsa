package webauthnadapter

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/go-webauthn/webauthn/webauthn"
)

func TestLoginChallengePayload_String(t *testing.T) {
	payload := loginChallengePayload{
		SessionData: webauthn.SessionData{
			Challenge: "secret-challenge-xyz",
		},
	}

	str := payload.String()

	if strings.Contains(str, "secret-challenge-xyz") {
		t.Errorf("loginChallengePayload did not redact SessionData.Challenge. Got: %s", str)
	}
	if !strings.Contains(str, "***REDACTED***") {
		t.Errorf("loginChallengePayload missing redaction marker. Got: %s", str)
	}
}

func TestChallengePayload_String(t *testing.T) {
	payload := challengePayload{
		SessionData: webauthn.SessionData{
			Challenge: "secret-challenge-abc",
		},
		InviteID: uuid.New(),
		UserID:   uuid.New(),
	}

	str := payload.String()

	if strings.Contains(str, "secret-challenge-abc") {
		t.Errorf("challengePayload did not redact SessionData.Challenge. Got: %s", str)
	}
	if !strings.Contains(str, "***REDACTED***") {
		t.Errorf("challengePayload missing redaction marker. Got: %s", str)
	}
	if !strings.Contains(str, payload.InviteID.String()) {
		t.Errorf("challengePayload missing InviteID. Got: %s", str)
	}
	if !strings.Contains(str, payload.UserID.String()) {
		t.Errorf("challengePayload missing UserID. Got: %s", str)
	}
}
