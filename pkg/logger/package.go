package logger

import (
	"io"
	"log/slog"

	"github.com/knadh/koanf/v2"
	"github.com/samber/do/v2"
	"github.com/whicu/hsa/internal/config"
)

type Service struct {
	logger *slog.Logger
	closer io.Closer
}

func (s *Service) Shutdown() error { return s.closer.Close() }

func newService(i do.Injector) (*Service, error) {
	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, err
	}
	log, closer, err := Logger(cfg)
	if err != nil {
		return nil, wrapLoggerError(err)
	}
	return &Service{logger: log, closer: closer}, nil
}

func newConfig(i do.Injector) (Config, error) {
	k, err := do.Invoke[*koanf.Koanf](i)
	if err != nil {
		return Config{}, err
	}
	def := defaultCfg
	return config.GetConfig(k, "logger", &def)
}

func newSlogLogger(i do.Injector) (*slog.Logger, error) {
	svc, err := do.Invoke[*Service](i)
	if err != nil {
		return nil, err
	}
	return svc.logger, nil
}

var Package = do.Package(
	do.Lazy(newConfig),
	do.Lazy(newService),
	do.Lazy(newSlogLogger),
)
