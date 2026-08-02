package user

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/whicu/hsa/internal/domain"
)

var (
	ErrIDRequired = errors.New("user: id is required")
	ErrInvitedBy  = errors.New("user: invited_by is required for non-root user")
)

type UserID = uuid.UUID

type User struct {
	id        UserID
	invitedBy *UserID
	createdAt time.Time
}

func New(id UserID, invitedBy UserID, createdAt time.Time) (u *User, err error) {
	defer func() {
		if err != nil {
			err = domain.ErrInvalidArgument(err)
		}
	}()
	if id == uuid.Nil {
		return nil, ErrIDRequired
	}
	if invitedBy == uuid.Nil {
		return nil, ErrInvitedBy
	}
	return &User{id: id, invitedBy: &invitedBy, createdAt: createdAt}, nil
}

func NewRoot(id UserID, createdAt time.Time) (u *User, err error) {
	defer func() {
		if err != nil {
			err = domain.ErrInvalidArgument(err)
		}
	}()
	if id == uuid.Nil {
		return nil, ErrIDRequired
	}
	return &User{id: id, invitedBy: nil, createdAt: createdAt}, nil
}

func (u *User) ID() UserID           { return u.id }
func (u *User) InvitedBy() *UserID   { return u.invitedBy }
func (u *User) IsRoot() bool         { return u.invitedBy == nil }
func (u *User) CreatedAt() time.Time { return u.createdAt }
