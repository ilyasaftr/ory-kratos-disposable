package sentry

import (
	"context"
	"log/slog"
	"os"

	sentryslog "github.com/getsentry/sentry-go/slog"
)

type multiHandler struct {
	handlers []slog.Handler
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, r.Level) {
			if err := handler.Handle(ctx, r); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithAttrs(attrs)
	}
	return &multiHandler{handlers: handlers}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		handlers[i] = handler.WithGroup(name)
	}
	return &multiHandler{handlers: handlers}
}

// NewHandler creates a new slog handler that combines stdout JSON logging
// with Sentry integration. When Sentry is enabled, logs are sent to both
// stdout and Sentry's Logs UI feature.
func NewHandler(cfg Config, minLevel slog.Level) (slog.Handler, error) {
	stdoutHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: false,
		Level:     minLevel,
	})

	// If Sentry is not enabled, return only stdout handler
	if !enabled {
		return stdoutHandler, nil
	}

	// Build level filters honoring minLevel
	var logLevels []slog.Level
	for _, lvl := range []slog.Level{slog.LevelDebug, slog.LevelInfo, slog.LevelWarn} {
		if lvl >= minLevel {
			logLevels = append(logLevels, lvl)
		}
	}

	eventLevels := []slog.Level{}
	if slog.LevelError >= minLevel {
		eventLevels = append(eventLevels, slog.LevelError)
	}
	if sentryslog.LevelFatal >= minLevel {
		eventLevels = append(eventLevels, sentryslog.LevelFatal)
	}

	// Configure Sentry handler with Logs UI support
	sentryHandler := sentryslog.Option{
		EventLevel: eventLevels,
		LogLevel:   logLevels,
	}.NewSentryHandler(context.Background())

	// Combine both handlers so logs go to both destinations
	return &multiHandler{
		handlers: []slog.Handler{stdoutHandler, sentryHandler},
	}, nil
}
