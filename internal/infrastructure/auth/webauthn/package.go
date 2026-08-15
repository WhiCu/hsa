package webauthnadapter

import (
	"fmt"
	"log/slog"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"
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
	return config.GetConfig(k, "webauthn", &def)
}

func newWebAuthn(i do.Injector) (*gowebauthn.WebAuthn, error) {
	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, fmt.Errorf("webauthnadapter: invoke config: %w", err)
	}

	wa, err := gowebauthn.New(&gowebauthn.Config{
		RPDisplayName: cfg.RP.DisplayName,
		RPID:          cfg.RP.ID,
		RPOrigins:     cfg.RP.Origins,
	})
	if err != nil {
		return nil, fmt.Errorf("webauthnadapter: init go-webauthn: %w", err)
	}

	return wa, nil
}

func newAuthenticator(i do.Injector) (*Authenticator, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, fmt.Errorf("webauthnadapter: invoke logger: %w", err)
	}

	wa, err := do.Invoke[*gowebauthn.WebAuthn](i)
	if err != nil {
		return nil, fmt.Errorf("webauthnadapter: invoke webauthn: %w", err)
	}

	challenge, err := do.InvokeAs[ChallengeCodec](i)
	if err != nil {
		return nil, fmt.Errorf("webauthnadapter: invoke challenge codec: %w", err)
	}

	creds, err := do.InvokeAs[CredentialsProvider](i)
	if err != nil {
		return nil, fmt.Errorf("webauthnadapter: invoke credentials provider: %w", err)
	}

	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, fmt.Errorf("webauthnadapter: invoke config: %w", err)
	}

	return NewAuthenticator(log, wa, challenge, creds, cfg.ChallengeTTL), nil
}

func newRegistrator(i do.Injector) (*Registrator, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, fmt.Errorf("webauthnadapter: invoke logger: %w", err)
	}

	wa, err := do.Invoke[*gowebauthn.WebAuthn](i)
	if err != nil {
		return nil, fmt.Errorf("webauthnadapter: invoke webauthn: %w", err)
	}

	challenge, err := do.InvokeAs[ChallengeCodec](i)
	if err != nil {
		return nil, fmt.Errorf("webauthnadapter: invoke challenge codec: %w", err)
	}

	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, fmt.Errorf("webauthnadapter: invoke config: %w", err)
	}

	return NewRegistrator(log, wa, challenge, cfg.ChallengeTTL), nil
}

var Package = do.Package(
	do.Lazy(newConfig),
	do.Lazy(newWebAuthn),
	do.Lazy(newAuthenticator),
	do.Lazy(newRegistrator),
)
