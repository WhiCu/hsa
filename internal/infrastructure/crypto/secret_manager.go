package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

type SecretManager struct {
	secretKey []byte
}

func NewSecretManager(secretKey []byte) *SecretManager {
	return &SecretManager{secretKey: secretKey}
}

// SECURITY: never log this field
func (sm *SecretManager) String() string {
	if sm == nil {
		return "<nil>"
	}
	return "SecretManager{secretKey: ***REDACTED***}"
}

func (sm *SecretManager) GenerateHash(raw string) (string, error) {
	return generateHashBase32(raw, sm.secretKey)
}

func (sm *SecretManager) GenerateToken(n int) (token, hash string, err error) {
	return generateToken(n, sm.secretKey)
}

func generateRandomBase32(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return b32.EncodeToString(buf), nil
}

func generateHash(raw string, secretKey []byte) ([]byte, error) {
	h := hmac.New(sha256.New, secretKey)
	_, err := h.Write([]byte(raw))
	if err != nil {
		return nil, err
	}
	sum := h.Sum(nil)
	return sum, nil
}

func generateHashBase32(raw string, secretKey []byte) (string, error) {
	sum, err := generateHash(raw, secretKey)
	if err != nil {
		return "", err
	}
	return b32.EncodeToString(sum), nil
}

func generateToken(n int, secretKey []byte) (raw, hash string, err error) {
	raw, err = generateRandomBase32(n)
	if err != nil {
		return "", "", err
	}
	hash, err = generateHashBase32(raw, secretKey)
	if err != nil {
		return "", "", err
	}
	return raw, hash, nil
}
