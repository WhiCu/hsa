package application

import (
	"context"
	"time"

	"github.com/whicu/hsa/internal/domain/session"
	"github.com/whicu/hsa/internal/domain/user"
)

const refreshTokenLength = 32

type RefreshTokenSaver interface {
	Save(ctx context.Context, t *session.RefreshToken) error
}

type TokenIssuer interface {
	IssueAccessToken(userID user.UserID, ttl time.Duration) (string, error)
}

type SessionIssuer struct {
	sessions      RefreshTokenSaver
	refreshTokens TokenGenerator
	accessTokens  TokenIssuer
	ids           IDGenerator
	refreshTTL    time.Duration
	accessTTL     time.Duration
}

func NewSessionIssuer(
	sessions RefreshTokenSaver,
	refreshTokens TokenGenerator,
	accessTokens TokenIssuer,
	ids IDGenerator,
	refreshTTL, accessTTL time.Duration,
) *SessionIssuer {
	return &SessionIssuer{sessions, refreshTokens, accessTokens, ids, refreshTTL, accessTTL}
}

func (si *SessionIssuer) Issue(ctx context.Context, userID user.UserID, deviceInfo, ipAddress string, now time.Time) (access, refresh string, err error) {
	rawRefresh, refreshHash, err := si.refreshTokens.GenerateToken(refreshTokenLength)
	if err != nil {
		return "", "", err
	}
	rt, err := session.New(si.ids.NewID(), userID, refreshHash, deviceInfo, ipAddress, si.refreshTTL, now)
	if err != nil {
		return "", "", err
	}
	if sessionsErr := si.sessions.Save(ctx, rt); sessionsErr != nil {
		return "", "", sessionsErr
	}
	access, err = si.accessTokens.IssueAccessToken(userID, si.accessTTL)
	if err != nil {
		return "", "", err
	}
	return access, rawRefresh, nil
}
