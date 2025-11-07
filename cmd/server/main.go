package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ilyasaftr/ory-kratos-disposable/internal/config"
	"github.com/ilyasaftr/ory-kratos-disposable/internal/handler"
	"github.com/ilyasaftr/ory-kratos-disposable/internal/logging"
	"github.com/ilyasaftr/ory-kratos-disposable/internal/middleware"
	"github.com/ilyasaftr/ory-kratos-disposable/internal/service"
	appSentry "github.com/ilyasaftr/ory-kratos-disposable/pkg/sentry"
)

// responseWriter is a wrapper for http.ResponseWriter that captures the status code
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// loggingMiddleware logs HTTP requests with their details
func loggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			rw := &responseWriter{
				ResponseWriter: w,
				status:         200, // default status
			}

			// Process request
			next.ServeHTTP(rw, r)

			// Log request completion
			duration := time.Since(start)
			logger.Info("request completed",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rw.status),
				slog.Int("size", rw.size),
				slog.Duration("duration", duration),
				slog.String("ip", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()))
		})
	}
}

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize Sentry (optional - disabled if DSN not provided)
	sentryConfig := appSentry.Config{
		DSN:              cfg.Sentry.DSN,
		Environment:      cfg.Sentry.Environment,
		SampleRate:       cfg.Sentry.SampleRate,
		TracesSampleRate: cfg.Sentry.TracesSampleRate,
		EnableTracing:    cfg.Sentry.EnableTracing,
		EnableLogs:       cfg.Sentry.EnableLogs,
		Debug:            cfg.Sentry.Debug,
	}
	if err := appSentry.Init(sentryConfig); err != nil {
		fmt.Fprintf(os.Stdout, "continuing without sentry integration: %v\n", err)
	}
	defer appSentry.Close(5 * time.Second)

	// Resolve log level from config
	minLevel := logging.ParseLevel(cfg.Logger.Level)

	// Create slog handler with Sentry integration
	logHandler, err := appSentry.NewHandler(sentryConfig, minLevel)
	if err != nil {
		fmt.Fprintf(os.Stdout, "failed to create sentry handler: %v, using stdout only\n", err)
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: false,
			Level:     minLevel,
		})
	}

	// Initialize slog with Sentry handler
	logger := slog.New(logHandler).With(
		slog.String("service", "ory-kratos-webhook"),
	)

	logger.Info("starting ory kratos disposable email webhook",
		slog.String("port", cfg.Server.Port),
		slog.Duration("refresh_interval", cfg.Refresh.Interval),
		slog.Int("list_urls_count", len(cfg.ListURLs)))

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize disposable email service
	disposableService := service.NewDisposableEmailService(
		cfg.ListURLs,
		cfg.Refresh.Interval,
		logger,
	)

	// Start the service (load initial data and start auto-refresh)
	// Note: Service always starts even if all URLs fail (fail mode)
	if err := disposableService.Start(ctx); err != nil {
		// This should never happen since Start() always returns nil, but keep for safety
		logger.Error("failed to start disposable email service", slog.Any("error", err))
		os.Exit(1)
	}

	// Initialize handlers
    validateHandler := handler.NewValidateHandler(disposableService, logger)
    healthHandler := handler.NewHealthHandler(disposableService, logger)

	// Initialize middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg.Webhook.APIKey, logger)

	// Setup HTTP router
	mux := http.NewServeMux()

	// Health check endpoint (no auth required)
	mux.HandleFunc("/health", healthHandler.Handle)

    // Validation endpoint (with auth)
    mux.HandleFunc("/v1/validate/email", authMiddleware.Authenticate(validateHandler.Handle))

	// Create HTTP handler with middleware chain
	var handler http.Handler = mux

	// Add Sentry HTTP middleware for panic recovery and error tracking
	handler = appSentry.HTTPMiddleware()(handler)

	// Add request logging middleware
	handler = loggingMiddleware(logger)(handler)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("server started", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")

	// Cancel context to stop background goroutines
	cancel()

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server forced to shutdown", slog.Any("error", err))
		os.Exit(1)
	}

	logger.Info("server stopped gracefully")
}
