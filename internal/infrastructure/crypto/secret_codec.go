package crypto

import (
	"encoding/json"
	"errors"
	"time"

	"aidanwoods.dev/go-paseto"
)

var (
	ErrTokenMalformed = errors.New("crypto: challenge token malformed")
	ErrTokenExpired   = errors.New("crypto: challenge token expired")
)

type SecretCodec struct {
	pasetoKey paseto.V4SymmetricKey
}

func NewSecretCodec(pasetoKey paseto.V4SymmetricKey) *SecretCodec {
	return &SecretCodec{pasetoKey: pasetoKey}
}

func (sc *SecretCodec) Encode(payload any, ttl time.Duration) (string, error) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	token := paseto.NewToken()
	token.SetExpiration(time.Now().Add(ttl))
	token.SetString("payload", string(raw))

	return token.V4Encrypt(sc.pasetoKey, nil), nil
}

func (sc *SecretCodec) Decode(tokenStr string, out any) error {
	parser := paseto.NewParser()

	token, err := parser.ParseV4Local(sc.pasetoKey, tokenStr, nil)
	if err != nil {
		if err, ok := errors.AsType[paseto.RuleError](err); ok {
			return errors.Join(ErrTokenExpired, err)
		}
		return ErrTokenMalformed
	}

	raw, err := token.GetString("payload")
	if err != nil {
		return ErrTokenMalformed
	}

	return json.Unmarshal([]byte(raw), out)
}
