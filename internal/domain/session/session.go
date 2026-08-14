package session

import (
	"errors"
	"fmt"
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/user"
)

var (
	ErrIDRequired        = errors.New("refresh_token: id is required")
	ErrTokenHashRequired = errors.New("refresh_token: token hash is required")
	ErrAlreadyRevoked    = errors.New("refresh_token: already revoked")
	ErrIPAddressRequired = errors.New("refresh_token: ip address is required")
)

type RefreshTokenID = uuid.UUID

type RefreshToken struct {
	id         RefreshTokenID
	userID     user.UserID
	tokenHash  string
	deviceInfo string
	ipAddress  netip.Addr
	expiresAt  time.Time
	revokedAt  *time.Time
	createdAt  time.Time
}

func New(
	id RefreshTokenID,
	userID user.UserID,
	tokenHash, deviceInfo string,
	ipAddress netip.Addr,
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
	if !ipAddress.IsValid() {
		return nil, ErrIPAddressRequired
	}
	return &RefreshToken{
		id:         id,
		userID:     userID,
		tokenHash:  tokenHash,
		deviceInfo: deviceInfo,
		ipAddress:  ipAddress,
		expiresAt:  now.Add(ttl),
		createdAt:  now,
	}, nil
}

func (t *RefreshToken) ID() RefreshTokenID    { return t.id }
func (t *RefreshToken) UserID() user.UserID   { return t.userID }
func (t *RefreshToken) TokenHash() string     { return t.tokenHash }
func (t *RefreshToken) DeviceInfo() string    { return t.deviceInfo }
func (t *RefreshToken) IPAddress() netip.Addr { return t.ipAddress }
func (t *RefreshToken) IsRevoked() bool       { return t.revokedAt != nil }
func (t *RefreshToken) ExpiresAt() time.Time  { return t.expiresAt }
func (t *RefreshToken) CreatedAt() time.Time  { return t.createdAt }
func (t *RefreshToken) RevokedAt() *time.Time { return t.revokedAt }

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

func (t *RefreshToken) Rotate(id RefreshTokenID, tokenHash string, now time.Time) (rotated *RefreshToken, err error) {
	defer func() {
		if err != nil {
			err = domain.ErrInvalidArgument(err)
		}
	}()
	if id == uuid.Nil {
		return nil, ErrIDRequired
	}
	if tokenHash == "" {
		return nil, ErrTokenHashRequired
	}
	return &RefreshToken{
		id: id, userID: t.userID, tokenHash: tokenHash,
		deviceInfo: t.deviceInfo, ipAddress: t.ipAddress,
		expiresAt: t.expiresAt,
		createdAt: now,
	}, nil
}

// SECURITY: never log this field
func (t *RefreshToken) String() string {
	if t == nil {
		return "<nil>"
	}
	return fmt.Sprintf("RefreshToken{id: %v, userID: %v, tokenHash: ***REDACTED***, deviceInfo: %v, ipAddress: %v, expiresAt: %v, revokedAt: %v, createdAt: %v}", t.id, t.userID, t.deviceInfo, t.ipAddress, t.expiresAt, t.revokedAt, t.createdAt)
}

func Reconstruct(id RefreshTokenID, userID user.UserID, tokenHash, deviceInfo string, ipAddress netip.Addr, expiresAt time.Time, revokedAt *time.Time, createdAt time.Time) *RefreshToken {
	return &RefreshToken{id: id, userID: userID, tokenHash: tokenHash, deviceInfo: deviceInfo, ipAddress: ipAddress, expiresAt: expiresAt, revokedAt: revokedAt, createdAt: createdAt}
}
