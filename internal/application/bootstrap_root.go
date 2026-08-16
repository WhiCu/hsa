// internal/application/bootstrap_root.go
package application

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/whicu/hsa/internal/domain"
	"github.com/whicu/hsa/internal/domain/user"
)

var ErrRootAlreadyExists = errors.New("application: root user already exists")

type RootFinder interface {
	FindRoot(ctx context.Context) (*user.User, error) // domain.ErrNotFound, если root ещё не создан
}

type BootstrapRoot struct {
	log    *slog.Logger
	users  UserSaver
	finder RootFinder
	ids    IDGenerator
}

func NewBootstrapRoot(log *slog.Logger, users UserSaver, finder RootFinder, ids IDGenerator) *BootstrapRoot {
	return &BootstrapRoot{log: log, users: users, finder: finder, ids: ids}
}

func (uc *BootstrapRoot) Execute(ctx context.Context) (*user.User, error) {
	existing, err := uc.finder.FindRoot(ctx)
	if err == nil {
		uc.log.WarnContext(ctx, "root already exists, skipping bootstrap", slog.String("root_id", existing.ID().String()))
		return nil, ErrRootAlreadyExists
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}

	now := time.Now()
	root, err := user.NewRoot(uc.ids.NewID(), now)
	if err != nil {
		return nil, err
	}

	if errSave := uc.users.Save(ctx, root); errSave != nil {
		return nil, errSave
	}

	uc.log.InfoContext(ctx, "root user created", slog.String("root_id", root.ID().String()))
	return root, nil
}
