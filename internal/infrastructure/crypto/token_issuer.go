package crypto

import (
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/whicu/hsa/internal/domain/user"
)

type AccessTokenIssuer struct {
	secretKey paseto.V4AsymmetricSecretKey
}

func NewAccessTokenIssuer(secretKey paseto.V4AsymmetricSecretKey) *AccessTokenIssuer {
	return &AccessTokenIssuer{secretKey: secretKey}
}

// SECURITY: never log this field
func (ti *AccessTokenIssuer) String() string {
	if ti == nil {
		return nilString
	}
	return "AccessTokenIssuer{secretKey: ***REDACTED***}"
}

func (ti *AccessTokenIssuer) IssueAccessToken(userID user.UserID, role user.Role, ttl time.Duration) (string, error) {
	token := paseto.NewToken()
	token.SetExpiration(time.Now().Add(ttl))
	token.SetString("user_id", userID.String())
	token.SetString("role", role.String())

	return token.V4Sign(ti.secretKey, nil), nil
}
