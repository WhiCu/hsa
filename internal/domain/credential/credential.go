package credential

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/user"
)

var (
	ErrIDRequired          = errors.New("credential: id is required")
	ErrExternalIDRequired  = errors.New("credential: external id is required")
	ErrPublicKeyRequired   = errors.New("credential: public key is required")
	ErrSignCountRegression = errors.New("credential: sign count regression detected")
	ErrAlreadyRevoked      = errors.New("credential: already revoked")
)

type CredentialID = uuid.UUID
type ExternalID = []byte

type Credential struct {
	id         CredentialID
	externalID ExternalID
	userID     user.UserID
	publicKey  []byte
	signCount  uint32
	transports []string
	createdAt  time.Time
	revokedAt  *time.Time
}

func New(
	id CredentialID,
	externalID ExternalID,
	userID user.UserID,
	publicKey []byte,
	transports []string,
	createdAt time.Time,
) (c *Credential, err error) {
	defer func() {
		if err != nil {
			err = domain.ErrInvalidArgument(err)
		}
	}()
	if id == uuid.Nil {
		return nil, ErrIDRequired
	}
	if len(externalID) == 0 {
		return nil, ErrExternalIDRequired
	}
	if userID == uuid.Nil {
		return nil, user.ErrIDRequired
	}
	if len(publicKey) == 0 {
		return nil, ErrPublicKeyRequired
	}
	return &Credential{
		id:         id,
		externalID: externalID,
		userID:     userID,
		publicKey:  publicKey,
		transports: transports,
		createdAt:  createdAt,
	}, nil
}

func (c *Credential) ID() CredentialID       { return c.id }
func (c *Credential) ExternalID() ExternalID { return c.externalID }
func (c *Credential) UserID() user.UserID    { return c.userID }
func (c *Credential) PublicKey() []byte      { return c.publicKey }
func (c *Credential) SignCount() uint32      { return c.signCount }
func (c *Credential) Transports() []string   { return c.transports }
func (c *Credential) CreatedAt() time.Time   { return c.createdAt }
func (c *Credential) IsRevoked() bool        { return c.revokedAt != nil }
func (c *Credential) RevokedAt() *time.Time  { return c.revokedAt }

func (c *Credential) Revoke(now time.Time) error {
	if c.revokedAt != nil {
		return ErrAlreadyRevoked
	}
	c.revokedAt = &now
	return nil
}
func (c *Credential) SetSignCount(n uint32) error {
	if n <= c.signCount && c.signCount != 0 {
		return ErrSignCountRegression
	}
	c.signCount = n
	return nil
}

// SECURITY: never log this field
func (c *Credential) String() string {
	if c == nil {
		return "<nil>"
	}
	return fmt.Sprintf("Credential{id: %v, externalID: ***REDACTED***, userID: %v, publicKey: ***REDACTED***, signCount: %d, transports: %v, createdAt: %v}", c.id, c.userID, c.signCount, c.transports, c.createdAt)
}
