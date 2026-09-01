package application

import (
	"context"
	"fmt"

	"github.com/whicu/hsa/internal/domain/credential"
	"github.com/whicu/hsa/internal/domain/invite"
	"github.com/whicu/hsa/internal/domain/user"
)

type RegistrationResult struct {
	UserID           user.UserID
	InviteID         invite.InviteID
	ExternalID       credential.ExternalID
	PublicKey        []byte
	Transports       []string
	InitialSignCount uint32
}

// SECURITY: never log this field
func (r RegistrationResult) String() string {
	return fmt.Sprintf("RegistrationResult{UserID: %v, InviteID: %v, CredentialID: %v, PublicKey: ***REDACTED***, Transports: %v, InitialSignCount: %d}", r.UserID, r.InviteID, r.ExternalID, r.Transports, r.InitialSignCount)
}

type Registrator interface {
	Begin(ctx context.Context, candidateUserID user.UserID, inviteID invite.InviteID) (challengeToken string, creationOptions []byte, err error)
	Finish(ctx context.Context, challengeToken string, response []byte) (RegistrationResult, error)
}

type AuthenticationResult struct {
	UserID       user.UserID
	ExternalID   []byte
	NewSignCount uint32
	CloneWarning bool
}

// SECURITY: never log this field
func (a AuthenticationResult) String() string {
	return fmt.Sprintf("AuthenticationResult{UserID: %v, ExternalID: ***REDACTED***, NewSignCount: %d}", a.UserID, a.NewSignCount)
}

type Authenticator interface {
	Begin(ctx context.Context) (challengeToken string, requestOptions []byte, err error)
	Finish(ctx context.Context, challengeToken string, response []byte) (AuthenticationResult, error)
}
