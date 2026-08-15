package crypto

type PASETOConfig struct {
	// SymmetricKey (hex) используется для шифрования challenge-токенов в TokenCodec (V4 Local).
	SymmetricKey string `koanf:"symmetric_key" validate:"required,hexadecimal"`

	// SecretKey (hex) используется для подписи Access токенов в AccessTokenIssuer (V4 Public).
	SecretKey string `koanf:"secret_key" validate:"required,hexadecimal"`

	// PublicKey (hex, опционально) используется для проверки Access токенов в AccessTokenVerifier.
	// Если не указан, автоматически выводится из SecretKey.
	PublicKey string `koanf:"public_key" validate:"omitempty,hexadecimal"`
}

type Config struct {
	// HMACSecret используется в SecretManager для хеширования токенов, кодов инвайтов и advisory locks.
	HMACSecret string       `koanf:"hmac_secret" validate:"required,min=32"`
	PASETO     PASETOConfig `koanf:"paseto" validate:"required"`
}

var defaultCfg = Config{}
