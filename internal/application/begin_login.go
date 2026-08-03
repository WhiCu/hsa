package application

import (
	"context"
	"log/slog"
)

type BeginLogin struct {
	log           *slog.Logger
	authenticator Authenticator
}

func NewBeginLogin(log *slog.Logger, authenticator Authenticator) *BeginLogin {
	return &BeginLogin{
		log:           log,
		authenticator: authenticator,
	}
}

func (uc *BeginLogin) Execute(ctx context.Context) (token string, opts []byte, err error) {
	uc.log.DebugContext(ctx, "executing begin login")

	token, opts, err = uc.authenticator.Begin(ctx)
	if err != nil {
		uc.log.ErrorContext(ctx, "failed to begin login authentication",
			slog.Any("error", err),
		)
		return "", nil, err
	}

	uc.log.InfoContext(ctx, "begin login completed successfully")

	return token, opts, nil
}
