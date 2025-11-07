package config

import (
	"fmt"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	Server   ServerConfig
	Webhook  WebhookConfig
	Logger   LoggerConfig
	Sentry   SentryConfig
	ListURLs []string `env:"DISPOSABLE_LIST_URLS" envSeparator:"," envDefault:"https://cdn.jsdelivr.net/gh/ilyasaftr/disposable-email-domains@main/lists/deny.txt"`
	Refresh  RefreshConfig
}

type ServerConfig struct {
	Port string `env:"WEBHOOK_PORT" envDefault:"8080"`
}

type WebhookConfig struct {
	APIKey string `env:"WEBHOOK_API_KEY,required"`
}

type LoggerConfig struct {
	Level string `env:"LOG_LEVEL" envDefault:"info"`
}

type RefreshConfig struct {
	Interval time.Duration `env:"DISPOSABLE_LIST_UPDATE_INTERVAL" envDefault:"30m"`
}

type SentryConfig struct {
	DSN              string  `env:"SENTRY_DSN"`                                 // If empty, Sentry is disabled
	Environment      string  `env:"SENTRY_ENVIRONMENT" envDefault:"production"` // e.g., "production", "development"
	SampleRate       float64 `env:"SENTRY_SAMPLE_RATE" envDefault:"1.0"`        // 0.0 to 1.0 for error sampling
	TracesSampleRate float64 `env:"SENTRY_TRACES_SAMPLE_RATE" envDefault:"0.1"` // 0.0 to 1.0 for performance monitoring
	EnableTracing    bool    `env:"SENTRY_ENABLE_TRACING" envDefault:"false"`   // Enable performance monitoring
	EnableLogs       bool    `env:"SENTRY_ENABLE_LOGS" envDefault:"false"`      // Enable Sentry Logs product for structured logging
	Debug            bool    `env:"SENTRY_DEBUG" envDefault:"false"`            // Enable debug mode for Sentry SDK
}

func Load() (*Config, error) {
	// Load .env file if it exists (ignore error if it doesn't)
	_ = godotenv.Load()

	// Parse environment variables into the config struct
	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return cfg, nil
}
