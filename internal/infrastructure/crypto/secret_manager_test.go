package crypto

import (
	"testing"
)

func TestSecretManager_String(t *testing.T) {
	sm := &SecretManager{secretKey: []byte("very-secret-key")}
	str := sm.String()

	if str != "SecretManager{secretKey: ***REDACTED***}" {
		t.Errorf("Expected string to be redacted, got: %s", str)
	}

	var nilSM *SecretManager
	if nilSM.String() != "<nil>" {
		t.Errorf("Expected <nil> for nil SecretManager, got: %s", nilSM.String())
	}
}
