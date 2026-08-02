package logger

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidLogLevel = errors.New("invalid log level")
)

func wrapLoggerError(err error) error {
	const prefix = "logger"
	return fmt.Errorf("%s: %w", prefix, err)
}
