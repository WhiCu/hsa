package webauthnadapter

import "time"

type RPConfig struct {
	ID          string   `koanf:"id" validate:"required"`
	DisplayName string   `koanf:"display_name" validate:"required"`
	Origins     []string `koanf:"origins" validate:"required,min=1,dive,required"`
}

type Config struct {
	RP           RPConfig      `koanf:"rp" validate:"required"`
	ChallengeTTL time.Duration `koanf:"challenge_ttl" validate:"required,gt=0"`
}

var defaultCfg = Config{
	ChallengeTTL: 5 * time.Minute,
}
