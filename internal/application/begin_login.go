package application

import "context"

type BeginLogin struct {
	authenticator Authenticator
}

func NewBeginLogin(authenticator Authenticator) *BeginLogin {
	return &BeginLogin{authenticator: authenticator}
}

func (uc *BeginLogin) Execute(ctx context.Context) (token string, opts []byte, err error) {
	token, opts, err = uc.authenticator.Begin(ctx)
	if err != nil {
		return "", nil, err
	}
	return token, opts, nil
}
