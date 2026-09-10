package application

import (
	"context"
	"strconv"
	"strings"

	"github.com/whicu/hsa/internal/domain/credential"
	"github.com/whicu/hsa/internal/domain/invite"
	"github.com/whicu/hsa/internal/domain/user"
)

type RegistrationResult struct {
	UserID   user.UserID
	InviteID invite.InviteID
	// SECURITY: never log this field
	ExternalID       credential.ExternalID
	PublicKey        []byte
	Transports       []string
	InitialSignCount uint32
}

// SECURITY: never log this field
func (r RegistrationResult) String() string {
	return "RegistrationResult{UserID: " + r.UserID.String() +
		", InviteID: " + r.InviteID.String() +
		", ExternalID: ***REDACTED***" +
		", PublicKey: ***REDACTED***" +
		", Transports: [" + strings.Join(r.Transports, " ") + "]" +
		", InitialSignCount: " + strconv.FormatUint(uint64(r.InitialSignCount), 10) + "}"
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
	return "AuthenticationResult{UserID: " + a.UserID.String() +
		", ExternalID: ***REDACTED***" +
		", NewSignCount: " + strconv.FormatUint(uint64(a.NewSignCount), 10) + "}"
}

type Authenticator interface {
	Begin(ctx context.Context) (challengeToken string, requestOptions []byte, err error)
	Finish(ctx context.Context, challengeToken string, response []byte) (AuthenticationResult, error)
}
