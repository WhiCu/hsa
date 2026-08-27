package invite

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/user"
)

type InviteID = uuid.UUID

var (
	ErrIDRequired       = errors.New("invite: id is required")
	ErrCodeHashRequired = errors.New("invite: code hash is required")
	ErrExpired          = errors.New("invite: expired")
	ErrAlreadyUsed      = errors.New("invite: already used")
	ErrTooManyActive    = errors.New("invite: active invite limit exceeded")
)

type Invite struct {
	id        InviteID
	createdBy user.UserID
	codeHash  string
	usedBy    *user.UserID
	usedAt    *time.Time
	expiresAt time.Time
	createdAt time.Time
}

func New(
	id InviteID,
	createdBy user.UserID,
	codeHash string,
	ttl time.Duration,
	now time.Time,
) (i *Invite, err error) {
	defer func() {
		if err != nil {
			err = domain.ErrInvalidArgument(err)
		}
	}()
	if id == uuid.Nil {
		return nil, ErrIDRequired
	}
	if createdBy == uuid.Nil {
		return nil, user.ErrIDRequired
	}
	if codeHash == "" {
		return nil, ErrCodeHashRequired
	}
	return &Invite{
		id: id, createdBy: createdBy, codeHash: codeHash,
		expiresAt: now.Add(ttl), createdAt: now,
	}, nil
}
func (i *Invite) ID() InviteID                 { return i.id }
func (i *Invite) CreatedBy() user.UserID       { return i.createdBy }
func (i *Invite) CodeHash() string             { return i.codeHash }
func (i *Invite) ExpiresAt() time.Time         { return i.expiresAt }
func (i *Invite) CreatedAt() time.Time         { return i.createdAt }
func (i *Invite) UsedBy() *user.UserID         { return i.usedBy }
func (i *Invite) UsedAt() *time.Time           { return i.usedAt }
func (i *Invite) IsExpired(now time.Time) bool { return now.After(i.expiresAt) }
func (i *Invite) IsUsed() bool                 { return i.usedAt != nil }

func (i *Invite) Redeem(by user.UserID, now time.Time) error {
	if i.IsUsed() {
		return ErrAlreadyUsed
	}
	if i.IsExpired(now) {
		return ErrExpired
	}

	i.usedBy = &by
	i.usedAt = &now

	return nil
}

// SECURITY: never log this field
func (i *Invite) String() string {
	if i == nil {
		return "<nil>"
	}
	usedByStr := "<nil>"
	if i.usedBy != nil {
		usedByStr = i.usedBy.String()
	}
	return fmt.Sprintf("Invite{id: %v, createdBy: %v, codeHash: ***REDACTED***, usedBy: %v, expiresAt: %v, createdAt: %v}", i.id, i.createdBy, usedByStr, i.expiresAt, i.createdAt)
}

func Reconstruct(id InviteID, createdBy user.UserID, codeHash string, usedBy *user.UserID, usedAt *time.Time, expiresAt, createdAt time.Time) *Invite {
	return &Invite{id: id, createdBy: createdBy, codeHash: codeHash, usedBy: usedBy, usedAt: usedAt, expiresAt: expiresAt, createdAt: createdAt}
}
