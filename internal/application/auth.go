package application

import (
	"context"

	"github.com/whicu/hsa/internal/domain/credential"
	"github.com/whicu/hsa/internal/domain/invite"
	"github.com/whicu/hsa/internal/domain/user"
)

type RegistrationResult struct {
	UserID           user.UserID
	InviteID         invite.InviteID
	CredentialID     credential.ExternalID
	PublicKey        []byte
	Transports       []string
	InitialSignCount uint32
}

type Registrator interface {
	Begin(ctx context.Context, candidateUserID user.UserID, inviteID invite.InviteID) (challengeToken string, creationOptions []byte, err error)
	Finish(ctx context.Context, challengeToken string, response []byte) (RegistrationResult, error)
}

type AuthenticationResult struct {
	UserID       user.UserID
	ExternalID   []byte
	NewSignCount uint32
}
type Authenticator interface {
	Begin(ctx context.Context) (challengeToken string, requestOptions []byte, err error)
	Finish(ctx context.Context, challengeToken string, response []byte) (AuthenticationResult, error)
}
