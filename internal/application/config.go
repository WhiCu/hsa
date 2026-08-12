package application

import "time"

type Config struct {
	Invite  InviteConfig  `koanf:"invite" validate:"required"`
	Session SessionConfig `koanf:"session" validate:"required"`
}

type InviteConfig struct {
	MaxActive int `koanf:"max_active" validate:"required,gt=0"`

	TTL time.Duration `koanf:"ttl" validate:"required,gt=0"`

	MaxWrappedKey int `koanf:"max_wrapped_key" validate:"required,gt=0"`
}

type SessionConfig struct {
	RefreshTTL time.Duration `koanf:"refresh_ttl" validate:"required,gt=0"`

	AccessTTL time.Duration `koanf:"access_ttl" validate:"required,gt=0"`
}

var defaultCfg = Config{
	Invite: InviteConfig{
		MaxActive: 3,
		TTL:       72 * time.Hour,
	},
	Session: SessionConfig{
		RefreshTTL: 30 * 24 * time.Hour,
		AccessTTL:  15 * time.Minute,
	},
}
