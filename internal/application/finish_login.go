package application

import (
	"context"
	"errors"
	"time"

	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/credential"
)

var ErrCredentialNotFound = errors.New("application: credential not found")

type CredentialFinder interface {
	FindByExternalID(ctx context.Context, externalID []byte) (*credential.Credential, error)
}

type Login struct {
	credentials     CredentialFinder
	credentialSaver CredentialSaver
	authenticator   Authenticator
	sessionIssuer   *SessionIssuer
	transactor      Transactor
}

func NewLogin(
	credentials CredentialFinder,
	credentialSaver CredentialSaver,
	authenticator Authenticator,
	sessionIssuer *SessionIssuer,
	transactor Transactor,
) *Login {
	return &Login{credentials, credentialSaver, authenticator, sessionIssuer, transactor}
}

type LoginInput struct {
	ChallengeToken         string
	AuthenticationResponse []byte
	DeviceInfo             string
	IPAddress              string
}

type LoginOutput struct {
	AccessToken  string
	RefreshToken string
}

func (uc *Login) Execute(ctx context.Context, in LoginInput) (*LoginOutput, error) {
	now := time.Now()

	result, err := uc.authenticator.Finish(ctx, in.ChallengeToken, in.AuthenticationResponse)
	if err != nil {
		return nil, err
	}

	var out LoginOutput

	err = uc.transactor.RunInTransaction(ctx, func(ctx context.Context) error {
		cred, txErr := uc.credentials.FindByExternalID(ctx, result.ExternalID)
		if txErr != nil {
			if errors.Is(txErr, domain.ErrNotFound) {
				return ErrCredentialNotFound
			}
			return txErr
		}

		cred.SetSignCount(result.NewSignCount)
		if csErr := uc.credentialSaver.Save(ctx, cred); csErr != nil {
			return csErr
		}

		access, refresh, txErr := uc.sessionIssuer.Issue(ctx, cred.UserID(), in.DeviceInfo, in.IPAddress, now)
		if txErr != nil {
			return txErr
		}

		out = LoginOutput{AccessToken: access, RefreshToken: refresh}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}
