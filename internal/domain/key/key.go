package key

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/credential"
	"github.com/whicu/hsa/internal/domain/user"
)

var (
	ErrIDRequired            = errors.New("wrapped_key: id is required")
	ErrScopeInvalid          = errors.New("wrapped_key: invalid dek scope")
	ErrWrappedDEKRequired    = errors.New("wrapped_key: wrapped dek is required")
	ErrWrapAlgorithmRequired = errors.New("wrapped_key: wrap algorithm is required")
)

type WrappedKeyID = uuid.UUID

type Scope byte

const (
	ScopeMain Scope = iota
	ScopeDecoy
)

func (s Scope) Valid() bool { return s == ScopeMain || s == ScopeDecoy }

type WrappedKey struct {
	id            WrappedKeyID
	userID        user.UserID
	credentialID  credential.CredentialID
	scope         Scope
	wrappedDEK    []byte
	wrapAlgorithm string
	createdAt     time.Time
}

func New(
	id WrappedKeyID,
	userID user.UserID,
	credentialID credential.CredentialID,
	scope Scope,
	wrappedDEK []byte,
	wrapAlgorithm string,
	createdAt time.Time,
) (w *WrappedKey, err error) {
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
	if credentialID == uuid.Nil {
		return nil, user.ErrIDRequired
	}
	if !scope.Valid() {
		return nil, ErrScopeInvalid
	}
	if len(wrappedDEK) == 0 {
		return nil, ErrWrappedDEKRequired
	}
	if wrapAlgorithm == "" {
		return nil, ErrWrapAlgorithmRequired
	}
	return &WrappedKey{
		id: id, userID: userID, credentialID: credentialID,
		scope: scope, wrappedDEK: wrappedDEK,
		wrapAlgorithm: wrapAlgorithm, createdAt: createdAt,
	}, nil
}

func (k *WrappedKey) ID() WrappedKeyID                      { return k.id }
func (k *WrappedKey) UserID() user.UserID                   { return k.userID }
func (k *WrappedKey) Scope() Scope                          { return k.scope }
func (k *WrappedKey) CredentialID() credential.CredentialID { return k.credentialID }
func (k *WrappedKey) WrappedDEK() []byte                    { return k.wrappedDEK }
func (k *WrappedKey) WrapAlgorithm() string                 { return k.wrapAlgorithm }
func (k *WrappedKey) CreatedAt() time.Time                  { return k.createdAt }

// SECURITY: never log this field
func (k *WrappedKey) String() string {
	if k == nil {
		return "<nil>"
	}
	return fmt.Sprintf("WrappedKey{id: %v, userID: %v, credentialID: %v, scope: %d, wrappedDEK: ***REDACTED***, wrapAlgorithm: %v}", k.id, k.userID, k.credentialID, k.scope, k.wrapAlgorithm)
}

func Reconstruct(
	id WrappedKeyID,
	userID user.UserID,
	credentialID credential.CredentialID,
	scope Scope,
	wrappedDEK []byte,
	wrapAlgorithm string,
	createdAt time.Time,
) *WrappedKey {
	return &WrappedKey{
		id:            id,
		userID:        userID,
		credentialID:  credentialID,
		scope:         scope,
		wrappedDEK:    wrappedDEK,
		wrapAlgorithm: wrapAlgorithm,
		createdAt:     createdAt,
	}
}
