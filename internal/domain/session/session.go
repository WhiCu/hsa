package session

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/user"
)

var (
	ErrIDRequired        = errors.New("refresh_token: id is required")
	ErrTokenHashRequired = errors.New("refresh_token: token hash is required")
	ErrAlreadyRevoked    = errors.New("refresh_token: already revoked")
)

type RefreshTokenID = uuid.UUID

type RefreshToken struct {
	id         RefreshTokenID
	userID     user.UserID
	tokenHash  string
	deviceInfo string
	ipAddress  string
	expiresAt  time.Time
	revokedAt  *time.Time
	createdAt  time.Time
}

func New(
	id RefreshTokenID,
	userID user.UserID,
	tokenHash, deviceInfo, ipAddress string,
	ttl time.Duration,
	now time.Time,
) (rt *RefreshToken, err error) {
	defer func() {
		if err != nil {
			err = domain.ErrInvalidArgument(err)
		}
	}()
	if id == uuid.Nil {
		return nil, ErrIDRequired
	}
	if userID == uuid.Nil {
		return nil, user.ErrIDRequired
	}
	if tokenHash == "" {
		return nil, ErrTokenHashRequired
	}
	return &RefreshToken{
		id: id, userID: userID, tokenHash: tokenHash,
		deviceInfo: deviceInfo, ipAddress: ipAddress,
		expiresAt: now.Add(ttl), createdAt: now,
	}, nil
}

func (t *RefreshToken) ID() RefreshTokenID {
	return t.id
}

func (t *RefreshToken) UserID() user.UserID {
	return t.userID
}

func (t *RefreshToken) IsValid(now time.Time) bool {
	return t.revokedAt == nil && now.Before(t.expiresAt)
}

func (t *RefreshToken) Revoke(now time.Time) error {
	if t.revokedAt != nil {
		return ErrAlreadyRevoked
	}
	t.revokedAt = &now
	return nil
}

// SECURITY: never log this field
func (t RefreshToken) String() string {
	return fmt.Sprintf("RefreshToken{id: %s, userID: %s, tokenHash: ***REDACTED***, deviceInfo: %s, ipAddress: %s, expiresAt: %s, revokedAt: %v, createdAt: %s}", t.id.String(), t.userID.String(), t.deviceInfo, t.ipAddress, t.expiresAt.String(), t.revokedAt, t.createdAt.String())
}
