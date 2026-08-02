package logger

import (
	"context"
	"log/slog"
)

type NOPHandler struct{}

func (NOPHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return false
}

func (NOPHandler) Handle(_ context.Context, _ slog.Record) error {
	return nil
}

func (h NOPHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return h
}

func (h NOPHandler) WithGroup(_ string) slog.Handler {
	return h
}

func NewNOPHandler() NOPHandler {
	return NOPHandler{}
}

func NewNOPSlog() *slog.Logger {
	return slog.New(NewNOPHandler())
}
