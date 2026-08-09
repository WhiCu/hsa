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

func (ti *AccessTokenIssuer) IssueAccessToken(userID user.UserID, ttl time.Duration) (string, error) {
	token := paseto.NewToken()
	token.SetExpiration(time.Now().Add(ttl))
	token.SetString("user_id", userID.String())

	return token.V4Sign(ti.secretKey, nil), nil
}
