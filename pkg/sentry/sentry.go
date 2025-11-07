package sentry

import (
	"fmt"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
)

var (
	enabled bool
)

// Config holds the Sentry configuration
type Config struct {
	DSN              string
	Environment      string
	SampleRate       float64
	TracesSampleRate float64
	EnableTracing    bool
	EnableLogs       bool
	Debug            bool
}

// Init initializes Sentry with the provided configuration.
// If DSN is empty, Sentry will be disabled and the function returns nil.
// This allows the application to run normally without Sentry when not configured.
func Init(cfg Config) error {
	// If DSN is not provided, disable Sentry
	if cfg.DSN == "" {
		enabled = false
		fmt.Fprintf(os.Stdout, "sentry disabled: no DSN provided\n")
		return nil
	}

	// Initialize Sentry client
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.DSN,
		Environment:      cfg.Environment,
		SampleRate:       cfg.SampleRate,
		TracesSampleRate: cfg.TracesSampleRate,
		EnableTracing:    cfg.EnableTracing,
		EnableLogs:       cfg.EnableLogs,
		Debug:            cfg.Debug,
		AttachStacktrace: true,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// You can modify or filter events here before they're sent
			return event
		},
	})

	if err != nil {
		enabled = false
		fmt.Fprintf(os.Stdout, "failed to initialize sentry: %v, continuing without error tracking\n", err)
		return err
	}

	enabled = true
	fmt.Fprintf(os.Stdout, "sentry initialized successfully: environment=%s sample_rate=%.2f tracing_enabled=%v logs_enabled=%v\n",
		cfg.Environment, cfg.SampleRate, cfg.EnableTracing, cfg.EnableLogs)

	return nil
}

func IsEnabled() bool {
	return enabled
}

// Close flushes any pending events and closes the Sentry client.
func Close(timeout time.Duration) {
	if !enabled {
		return
	}

	sentry.Flush(timeout)
}

// CaptureException captures an exception and sends it to Sentry.
func CaptureException(err error) {
	if !enabled || err == nil {
		return
	}

	sentry.CaptureException(err)
}

// CaptureMessage captures a message and sends it to Sentry.
func CaptureMessage(message string) {
	if !enabled || message == "" {
		return
	}

	sentry.CaptureMessage(message)
}

// WithScope allows you to execute code with a custom Sentry scope.
func WithScope(f func(scope *sentry.Scope)) {
	if !enabled {
		return
	}

	sentry.WithScope(f)
}
