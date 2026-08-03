package credential

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/user"
)

var (
	ErrIDRequired         = errors.New("credential: id is required")
	ErrExternalIDRequired = errors.New("credential: external id is required")
	ErrPublicKeyRequired  = errors.New("credential: public key is required")
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
		id: id, userID: userID, publicKey: publicKey,
		transports: transports, createdAt: createdAt,
	}, nil
}

func (c *Credential) ID() CredentialID       { return c.id }
func (c *Credential) ExternalID() ExternalID { return c.externalID }
func (c *Credential) UserID() user.UserID    { return c.userID }
func (c *Credential) SignCount() uint32      { return c.signCount }

func (c *Credential) SetSignCount(n uint32) { c.signCount = n }
