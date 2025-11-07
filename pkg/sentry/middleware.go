package sentry

import (
	"net/http"
	"time"

	sentryhttp "github.com/getsentry/sentry-go/http"
)

func HTTPMiddleware() func(http.Handler) http.Handler {
	if !enabled {
		// Sentry not enabled
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	// Create Sentry HTTP handler with configuration
	sentryHandler := sentryhttp.New(sentryhttp.Options{
		Repanic:         false,
		WaitForDelivery: true,
		Timeout:         2 * time.Second,
	})

	return func(next http.Handler) http.Handler {
		return sentryHandler.Handle(next)
	}
}
