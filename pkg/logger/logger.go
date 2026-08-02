package logger

import (
	"io"
	"log/slog"
	"strings"

	phuslog "github.com/phuslu/log"
	phuslogotel "github.com/phuslu/log/otel"
	"go.opentelemetry.io/contrib/bridges/otelslog"
)

func GetSubLogger(logger *slog.Logger, group string) *slog.Logger {
	return logger.WithGroup(group)
}
func Logger(cfg Config) (*slog.Logger, io.Closer, error) {
	level, err := levelWrapper(cfg.Level)
	if err != nil {
		return nil, nil, err
	}

	pLog := phuslog.Logger{
		Level:  level,
		Caller: cfg.Caller,
		Writer: LogWriter(cfg.File),
	}
	phusProvider := &phuslogotel.LoggerProvider{Log: pLog}

	consoleHandler := otelslog.NewHandler("logger", otelslog.WithLoggerProvider(phusProvider))

	otlpHandler := otelslog.NewHandler("logger")

	handler := slog.NewMultiHandler(consoleHandler, otlpHandler)
	closer, ok := pLog.Writer.(io.Closer)
	if !ok {
		panic("logger writer does not implement io.Closer")
	}
	return slog.New(handler), closer, nil
}

func LogWriter(cfg FileConfig) phuslog.Writer {
	return &phuslog.MultiEntryWriter{
		&phuslog.ConsoleWriter{
			ColorOutput:    true,
			QuoteString:    true,
			EndWithMessage: true,
		},
		&phuslog.AsyncWriter{
			ChannelSize:   cfg.ChannelSize,
			DiscardOnFull: cfg.Discard,
			Writer: &phuslog.FileWriter{
				Filename:     cfg.Name,
				EnsureFolder: true,

				MaxSize:    cfg.Size,
				FileMode:   0600,
				MaxBackups: cfg.Backups,
				LocalTime:  false,
			},
		},
	}
}

func levelWrapper(level string) (phuslog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "trace":
		return phuslog.TraceLevel, nil
	case "debug":
		return phuslog.DebugLevel, nil
	case "info":
		return phuslog.InfoLevel, nil
	case "warn":
		return phuslog.WarnLevel, nil
	case "error":
		return phuslog.ErrorLevel, nil
	case "fatal":
		return phuslog.FatalLevel, nil
	case "panic":
		return phuslog.PanicLevel, nil
	default:
		return 0, ErrInvalidLogLevel
	}
}
