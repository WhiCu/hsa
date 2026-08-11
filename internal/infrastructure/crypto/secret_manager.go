package crypto

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"errors"
	"hash"
	"sync"
	"unsafe"
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

type hmacHolder struct {
	h   hash.Hash
	buf [sha256.Size]byte
}

type SecretManager struct {
	secretKey []byte
	hmacPool  sync.Pool
}

func NewSecretManager(secretKey []byte) *SecretManager {
	return &SecretManager{
		secretKey: secretKey,
		hmacPool: sync.Pool{
			New: func() any {
				return &hmacHolder{
					h: hmac.New(sha256.New, secretKey),
				}
			},
		},
	}
}

// SECURITY: never log this field
func (sm *SecretManager) String() string {
	if sm == nil {
		return nilString
	}
	return "SecretManager{secretKey: ***REDACTED***}"
}

func (sm *SecretManager) GenerateHash(raw string) (string, error) {
	return sm.generateHashBase32(raw)
}

func (sm *SecretManager) GenerateToken(n int) (token, hash string, err error) {
	return sm.generateToken(n)
}

func generateRandomBase32(n int) (string, error) {
	if n < 0 || n > 1<<20 {
		return "", errors.New("crypto: invalid length")
	}
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return b32.EncodeToString(buf), nil
}

func (sm *SecretManager) generateHashBase32(raw string) (string, error) {
	v := sm.hmacPool.Get()
	item, ok := v.(*hmacHolder)
	if !ok {
		return "", errors.New("crypto: type assertion failed on hash pool")
	}
	defer sm.hmacPool.Put(item)
	item.h.Reset()

	_, err := item.h.Write(stringToBytes(raw))
	if err != nil {
		return "", err
	}
	sum := item.h.Sum(item.buf[:0])
	encoded := b32.EncodeToString(sum)
	return encoded, nil
}

func (sm *SecretManager) generateToken(n int) (raw, hash string, err error) {
	raw, err = generateRandomBase32(n)
	if err != nil {
		return "", "", err
	}
	hash, err = sm.generateHashBase32(raw)
	if err != nil {
		return "", "", err
	}
	return raw, hash, nil
}

func stringToBytes(s string) []byte {
	if len(s) == 0 {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
