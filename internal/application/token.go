package application

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

func generateRandomBase32(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return b32.EncodeToString(buf), nil
}

func generateToken(n int) (raw, hash string, err error) {
	raw, err = generateRandomBase32(n)
	if err != nil {
		return "", "", err
	}
	return raw, generateHash(raw), nil
}

func generateHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return b32.EncodeToString(sum[:])
}
