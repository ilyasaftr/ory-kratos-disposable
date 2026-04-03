package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/ilyasaftr/ory-kratos-disposable/internal/config"
	"github.com/ilyasaftr/ory-kratos-disposable/internal/handler"
	"github.com/ilyasaftr/ory-kratos-disposable/internal/logging"
	"github.com/ilyasaftr/ory-kratos-disposable/internal/middleware"
	"github.com/ilyasaftr/ory-kratos-disposable/internal/service"
	appSentry "github.com/ilyasaftr/ory-kratos-disposable/pkg/sentry"
)

const (
	sentryFlushTimeout = 5 * time.Second
	shutdownTimeout    = 30 * time.Second
)

type App struct {
	server            *http.Server
	logger            *slog.Logger
	disposableService *service.DisposableEmailService
	cfg               *config.Config
}

func New(cfg *config.Config) *App {
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

	minLevel := logging.ParseLevel(cfg.Logger.Level)
	logHandler, err := appSentry.NewHandler(sentryConfig, minLevel)
	if err != nil {
		fmt.Fprintf(os.Stdout, "failed to create sentry handler: %v, using stdout only\n", err)
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			AddSource: false,
			Level:     minLevel,
		})
	}

	logger := slog.New(logHandler).With(slog.String("service", "ory-kratos-webhook"))
	disposableService := service.NewDisposableEmailService(
		cfg.ListURLs,
		cfg.Refresh.Interval,
		cfg.Failure.Mode,
		logger,
	)

	validateHandler := handler.NewValidateHandler(disposableService, logger)
	healthHandler := handler.NewHealthHandler(disposableService, logger)
	authMiddleware := middleware.NewAuthMiddleware(cfg.Webhook.APIKey, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler.Handle)
	mux.HandleFunc("/v1/validate/email", authMiddleware.Authenticate(validateHandler.Handle))

	var rootHandler http.Handler = mux
	rootHandler = appSentry.HTTPMiddleware()(rootHandler)
	rootHandler = middleware.NewRequestLoggingMiddleware(logger)(rootHandler)

	server := &http.Server{
		Addr:         ":" + cfg.Server.Port,
		Handler:      rootHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &App{
		server:            server,
		logger:            logger,
		disposableService: disposableService,
		cfg:               cfg,
	}
}

func (a *App) Run(ctx context.Context) error {
	a.logger.Info("starting ory kratos disposable email webhook",
		slog.String("port", a.cfg.Server.Port),
		slog.Duration("refresh_interval", a.cfg.Refresh.Interval),
		slog.Int("list_urls_count", len(a.cfg.ListURLs)))

	serviceCtx, cancelService := context.WithCancel(context.Background())
	defer cancelService()

	if err := a.disposableService.Start(serviceCtx); err != nil {
		return fmt.Errorf("failed to start disposable email service: %w", err)
	}

	serverErrCh := make(chan error, 1)
	go func() {
		a.logger.Info("server started", slog.String("addr", a.server.Addr))
		if err := a.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErrCh <- err
		}
		close(serverErrCh)
	}()

	select {
	case err := <-serverErrCh:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	case <-ctx.Done():
		a.logger.Info("shutting down server...")
	}

	cancelService()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("server forced to shutdown: %w", err)
	}

	a.logger.Info("server stopped gracefully")
	return nil
}

func (a *App) Close() {
	appSentry.Close(sentryFlushTimeout)
}
