package storage

import (
	"context"
	"log/slog"

	"github.com/knadh/koanf/v2"
	"github.com/samber/do/v2"
	"github.com/whicu/hsa/internal/config"
)

func newConfig(i do.Injector) (Config, error) {
	k, err := do.Invoke[*koanf.Koanf](i)
	if err != nil {
		return Config{}, err
	}
	def := defaultCfg
	return config.GetConfig(k, "logger", &def)
}

func newStorage(i do.Injector) (*Storage, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, err
	}

	ctx, err := do.Invoke[context.Context](i)
	if err != nil {
		return nil, err
	}

	return NewStorage(ctx, log, cfg)
}

func newCredentialRepository(i do.Injector) (*CredentialRepository, error) {
	storage, err := do.Invoke[*Storage](i)
	if err != nil {
		return nil, err
	}

	return NewCredentialRepository(storage), nil
}

func newInviteRepository(i do.Injector) (*InviteRepository, error) {
	storage, err := do.Invoke[*Storage](i)
	if err != nil {
		return nil, err
	}

	return NewInviteRepository(storage), nil
}

func newSessionRepository(i do.Injector) (*SessionRepository, error) {
	storage, err := do.Invoke[*Storage](i)
	if err != nil {
		return nil, err
	}

	return NewSessionRepository(storage), nil
}

func newUserRepository(i do.Injector) (*UserRepository, error) {
	storage, err := do.Invoke[*Storage](i)
	if err != nil {
		return nil, err
	}

	return NewUserRepository(storage), nil
}

func newWrappedKeyRepository(i do.Injector) (*WrappedKeyRepository, error) {
	storage, err := do.Invoke[*Storage](i)
	if err != nil {
		return nil, err
	}

	return NewWrappedKeyRepository(storage), nil
}

var Package = do.Package(
	do.Lazy(newConfig),
	do.Lazy(newStorage),
	do.Lazy(newCredentialRepository),
	do.Lazy(newInviteRepository),
	do.Lazy(newSessionRepository),
	do.Lazy(newUserRepository),
	do.Lazy(newWrappedKeyRepository),
)
