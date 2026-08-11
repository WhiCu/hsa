package crypto

import (
	"errors"

	"aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
	"github.com/whicu/hsa/internal/domain/user"
)

type AccessTokenVerifier struct {
	publicKey paseto.V4AsymmetricPublicKey
}

func NewAccessTokenVerifier(publicKey paseto.V4AsymmetricPublicKey) *AccessTokenVerifier {
	return &AccessTokenVerifier{publicKey: publicKey}
}

// SECURITY: never log this field
func (tv *AccessTokenVerifier) String() string {
	if tv == nil {
		return "<nil>"
	}
	return "AccessTokenVerifier{publicKey: ***REDACTED***}"
}

func (tv *AccessTokenVerifier) Verify(tokenStr string) (user.UserID, error) {
	parser := paseto.NewParser()

	token, err := parser.ParseV4Public(tv.publicKey, tokenStr, nil)
	if err != nil {
		if err, ok := errors.AsType[paseto.RuleError](err); ok {
			return user.UserID{}, errors.Join(ErrTokenExpired, err)
		}
		return user.UserID{}, ErrTokenMalformed
	}

	raw, err := token.GetString("user_id")
	if err != nil {
		return user.UserID{}, ErrTokenMalformed
	}

	userID, err := uuid.Parse(raw)
	if err != nil {
		return user.UserID{}, ErrTokenMalformed
	}

	return userID, nil
}
