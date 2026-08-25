package di

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/whicu/hsa/internal/domain/user"

	"github.com/knadh/koanf/v2"
	"github.com/samber/do/v2"
	"github.com/whicu/hsa/internal/application"
	"github.com/whicu/hsa/internal/config"
	webauthnadapter "github.com/whicu/hsa/internal/infrastructure/auth/webauthn"
	"github.com/whicu/hsa/internal/infrastructure/crypto"
	"github.com/whicu/hsa/internal/infrastructure/storage"
	"github.com/whicu/hsa/internal/infrastructure/telemetry"
	"github.com/whicu/hsa/pkg/idgen"
	"github.com/whicu/hsa/pkg/logger"
)

type Config struct {
	CfgView     bool
	RootInvites struct {
		CountInvites int
		Out          io.Writer
	}
}

func New(ctx context.Context, configPath string) *do.RootScope {
	injector := do.NewWithOpts(&do.InjectorOpts{
		Logf:                     diLogf,
		HealthCheckParallelism:   16,
		HealthCheckGlobalTimeout: 20 * time.Second,
	})

	do.ProvideValue(injector, ctx)
	registerPackages(injector, configPath)

	return injector
}

func diLogf(format string, args ...any) {
	fmt.Printf("[DI] "+format+"\n", args...)
}

func registerPackages(i do.Injector, configPath string) {
	config.Package(configPath)(i) // no dependencies
	idgen.Package(i)              // no dependencies
	telemetry.Package(i)          // config
	logger.Package(i)             // config, telemetry
	storage.Package(i)            // config, telemetry
	crypto.Package(i)             // config
	webauthnadapter.Package(i)    // config, crypto, storage, telemetry
	application.Package(i)        // config, webauthnadapter, crypto, storage, telemetry
}

func Run(ctx context.Context, injector *do.RootScope, cfg Config) error {
	if cfg.CfgView {
		if err := cfgView(injector); err != nil {
			return fmt.Errorf("cfg view: %w", err)
		}
	}

	if err := initTelemetry(injector); err != nil {
		return err
	}

	log, err := initLogger(injector)
	if err != nil {
		return err
	}

	if errSrg := initStorage(ctx, injector); errSrg != nil {
		return errSrg
	}

	root, errBootstrap := bootstrapRoot(ctx, injector, log)
	if errBootstrap != nil {
		return errBootstrap
	}

	rootCfg := cfg.RootInvites
	if rootCfg.CountInvites > 0 {
		if errRootInvites := createRootInvites(ctx, injector, root, rootCfg.CountInvites, rootCfg.Out); errRootInvites != nil {
			return errRootInvites
		}
	}

	return waitForShutdown(ctx, injector)
}

func initTelemetry(injector do.Injector) error {
	if _, err := do.Invoke[*telemetry.Service](injector); err != nil {
		return fmt.Errorf("init telemetry service: %w", err)
	}
	return nil
}

func initLogger(injector do.Injector) (*slog.Logger, error) {
	log, err := do.Invoke[*slog.Logger](injector)
	if err != nil {
		return nil, fmt.Errorf("init logger: %w", err)
	}
	return log, nil
}

func initStorage(ctx context.Context, injector do.Injector) error {
	srg, err := do.Invoke[*storage.Storage](injector)
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	if errUp := srg.Up(ctx); errUp != nil {
		return fmt.Errorf("storage up: %w", errUp)
	}
	return nil
}

func bootstrapRoot(ctx context.Context, injector do.Injector, log *slog.Logger) (*user.User, error) {
	bootstrap, err := do.Invoke[*application.BootstrapRoot](injector)
	if err != nil {
		return nil, fmt.Errorf("init bootstrap root: %w", err)
	}

	root, err := bootstrap.Execute(ctx)
	if err != nil && !errors.Is(err, application.ErrRootAlreadyExists) {
		return nil, fmt.Errorf("bootstrap root: %w", err)
	}

	log.InfoContext(ctx, "root user successfully created/found", slog.String("root_id", root.ID().String()))
	return root, nil
}

type RootInvite struct {
	Code      string    `json:"code"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func createRootInvites(ctx context.Context, injector do.Injector, root *user.User, n int, out io.Writer) error {
	rootCreateInvite, err := do.Invoke[*application.RootCreateInvite](injector)
	if err != nil {
		return fmt.Errorf("init root create invite: %w", err)
	}

	invites := make([]RootInvite, n)
	for i := range n {
		code, expiresAt, errExec := rootCreateInvite.Execute(ctx, root.ID())
		if errExec != nil {
			return fmt.Errorf("create root's invites: %w", errExec)
		}
		invites[i] = RootInvite{Code: code, ExpiresAt: expiresAt}
	}

	if errPrint := printInvitesJSON(invites, out); errPrint != nil {
		return fmt.Errorf("write invites to json: %w", errPrint)
	}
	return nil
}

func printInvitesJSON(invites []RootInvite, out io.Writer) error {
	err := json.MarshalWrite(
		out,
		invites,
		jsontext.WithIndent("  "),
	)
	if err != nil {
		return err
	}
	fmt.Fprintln(out)

	return nil
}

func waitForShutdown(ctx context.Context, injector *do.RootScope) error {
	_, report := injector.ShutdownOnSignalsWithContext(ctx)
	if errStr := report.Error(); errStr != "" {
		return fmt.Errorf("shutdown: %v", errStr)
	}
	return nil
}

func cfgView(i do.Injector) error {
	k, err := do.Invoke[*koanf.Koanf](i)
	if err != nil {
		return fmt.Errorf("init koanf: %w", err)
	}
	fmt.Println(config.DumpFlat(k))
	return nil
}
