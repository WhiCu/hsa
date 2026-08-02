package telemetry

import "time"

type Config struct {
	ServiceName    string `koanf:"service_name" validate:"required"`
	ServiceVersion string `koanf:"service_version" validate:"required,semver"`
	Environment    string `koanf:"environment" validate:"required,oneof=dev stage prod"`
	Enabled        bool   `koanf:"enabled"`

	Exporter ExporterConfig `koanf:"exporter" validate:"required"`

	Sampler SamplerConfig `koanf:"sampler" validate:"required"`

	Metric MetricConfig `koanf:"metric" validate:"required"`
}

type ExporterConfig struct {
	Conn ConnConfig `koanf:"conn" validate:"required"`

	Timeout time.Duration `koanf:"timeout" validate:"required,gt=0"`
}

type ConnConfig struct {
	Endpoint   string `koanf:"endpoint" validate:"required,hostname_port"`
	MaxMsgSize int    `koanf:"max_msg_size" validate:"min=0"`
	Insecure   bool   `koanf:"insecure"`
	Compressor string `koanf:"compressor" validate:"oneof=gzip ''"`

	KeepAlive KeepAliveConfig `koanf:"keep_alive" validate:"required"`

	Backoff BackoffConfig `koanf:"backoff" validate:"required"`
}

type KeepAliveConfig struct {
	Time time.Duration `koanf:"time" validate:"required,gt=0"`

	Timeout time.Duration `koanf:"timeout" validate:"required,gt=0"`

	PermitWithoutStream bool `koanf:"permit_without_stream"`
}

type BackoffConfig struct {
	BaseDelay time.Duration `koanf:"base_delay" validate:"required,gt=0"`

	MaxDelay time.Duration `koanf:"max_delay" validate:"required,gt=0"`

	Multiplier float64 `koanf:"multiplier" validate:"gte=0"`

	Jitter float64 `koanf:"jitter" validate:"gte=0,lte=1"`
}

type SamplerConfig struct {
	Type string `koanf:"type" validate:"required,oneof=always never ratio"`

	Ratio float64 `koanf:"ratio" validate:"gte=0,lte=1"`
}

type MetricConfig struct {
	Interval time.Duration `koanf:"interval" validate:"required,gt=0"`
}

var defaultCfg = Config{
	ServiceName:    "unknown-service",
	ServiceVersion: "0.1.0",
	Environment:    "dev",
	Enabled:        true,
	Exporter: ExporterConfig{
		Conn: ConnConfig{
			Endpoint:   "127.0.0.1:4317",
			MaxMsgSize: 16 * 1024 * 1024, // 16MB
			Insecure:   true,
			Compressor: "gzip",
			KeepAlive: KeepAliveConfig{
				Time:                30 * time.Second,
				Timeout:             5 * time.Second,
				PermitWithoutStream: true,
			},
			Backoff: BackoffConfig{
				BaseDelay:  1 * time.Second,
				MaxDelay:   30 * time.Second,
				Multiplier: 1.6,
				Jitter:     0.2,
			},
		},
		Timeout: 5 * time.Second,
	},
	Sampler: SamplerConfig{
		Type: "always",
	},
	Metric: MetricConfig{
		Interval: 15 * time.Second,
	},
}
