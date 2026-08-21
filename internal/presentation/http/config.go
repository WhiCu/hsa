package http

import (
	"net"
	"strconv"
	"time"
)

type CORSConfig struct {
	AllowedOrigins []string `koanf:"allowed_origins" validate:"required,min=1"`
}

type Config struct {
	Host string `koanf:"host"`
	Port uint16 `koanf:"port"`

	ReadTimeout       time.Duration `koanf:"read_timeout"`
	ReadHeaderTimeout time.Duration `koanf:"read_header_timeout"`
	WriteTimeout      time.Duration `koanf:"write_timeout"`
	IdleTimeout       time.Duration `koanf:"idle_timeout"`
	ShutdownTimeout   time.Duration `koanf:"shutdown_timeout"`

	MaxHeaderBytes   int   `koanf:"max_header_bytes"`
	RequestSizeLimit int64 `koanf:"max_body_bytes"`

	TrustedProxies []string   `koanf:"trusted_proxies"`
	CORS           CORSConfig `koanf:"cors" validate:"required"`
}

func (c Config) HostPort() string { return net.JoinHostPort(c.Host, strconv.Itoa(int(c.Port))) }

var defaultCfg = Config{
	Host: "0.0.0.0",
	Port: 8080,

	ReadTimeout:       15 * time.Second,
	ReadHeaderTimeout: 5 * time.Second,
	WriteTimeout:      15 * time.Second,
	IdleTimeout:       60 * time.Second,
	ShutdownTimeout:   10 * time.Second,

	MaxHeaderBytes:   1 << 20, // 1 MiB
	RequestSizeLimit: 2 << 20, // 2 MiB
}
