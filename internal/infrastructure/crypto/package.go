package crypto

import (
	"fmt"

	"aidanwoods.dev/go-paseto"
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
	return config.GetConfig(k, "crypto", &def)
}

func newSecretManager(i do.Injector) (*SecretManager, error) {
	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, fmt.Errorf("crypto: invoke config: %w", err)
	}

	return NewSecretManager([]byte(cfg.HMACSecret)), nil
}

func newTokenCodec(i do.Injector) (*TokenCodec, error) {
	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, fmt.Errorf("crypto: invoke config: %w", err)
	}

	symKey, err := paseto.V4SymmetricKeyFromHex(cfg.PASETO.SymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("crypto: parse paseto symmetric key: %w", err)
	}

	return NewTokenCodec(symKey), nil
}

func newAccessTokenIssuer(i do.Injector) (*AccessTokenIssuer, error) {
	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, fmt.Errorf("crypto: invoke config: %w", err)
	}

	secretKey, err := paseto.NewV4AsymmetricSecretKeyFromHex(cfg.PASETO.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("crypto: parse paseto secret key: %w", err)
	}

	return NewAccessTokenIssuer(secretKey), nil
}

func newAccessTokenVerifier(i do.Injector) (*AccessTokenVerifier, error) {
	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, fmt.Errorf("crypto: invoke config: %w", err)
	}

	// 1. Если задан явный публичный ключ — парсим его
	if cfg.PASETO.PublicKey != "" {
		pubKey, errkey := paseto.NewV4AsymmetricPublicKeyFromHex(cfg.PASETO.PublicKey)
		if errkey != nil {
			return nil, fmt.Errorf("crypto: parse paseto public key: %w", errkey)
		}
		return NewAccessTokenVerifier(pubKey), nil
	}

	// 2. Иначе автоматически выводим публичный ключ из секретного ключа
	secretKey, err := paseto.NewV4AsymmetricSecretKeyFromHex(cfg.PASETO.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("crypto: derive paseto public key from secret key: %w", err)
	}

	return NewAccessTokenVerifier(secretKey.Public()), nil
}

var Package = do.Package(
	do.Lazy(newConfig),
	do.Lazy(newSecretManager),
	do.Lazy(newTokenCodec),
	do.Lazy(newAccessTokenIssuer),
	do.Lazy(newAccessTokenVerifier),
)
