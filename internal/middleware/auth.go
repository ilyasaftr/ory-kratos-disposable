package middleware

import (
	"crypto/subtle"
	"log/slog"
	"net/http"

	"github.com/ilyasaftr/ory-kratos-disposable/internal/httpx"
)

const apiKeyHeader = "X-API-Key"

type AuthMiddleware struct {
	apiKey string
	logger *slog.Logger
}

func NewAuthMiddleware(apiKey string, log *slog.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		apiKey: apiKey,
		logger: log,
	}
}

// Authenticate wraps a handler with API key authentication
func (m *AuthMiddleware) Authenticate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get(apiKeyHeader)

		if apiKey == "" {
			m.logger.Warn("missing API key",
				slog.String("path", r.URL.Path),
				slog.String("method", r.Method),
				slog.String("ip", r.RemoteAddr))
			if err := httpx.WriteOryError(w, http.StatusUnauthorized, "Missing API key"); err != nil {
				m.logger.Error("failed to write auth error response", slog.Any("error", err))
			}
			return
		}

		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(m.apiKey)) != 1 {
			m.logger.Warn("invalid API key",
				slog.String("path", r.URL.Path),
				slog.String("method", r.Method),
				slog.String("ip", r.RemoteAddr))
			if err := httpx.WriteOryError(w, http.StatusUnauthorized, "Invalid API key"); err != nil {
				m.logger.Error("failed to write auth error response", slog.Any("error", err))
			}
			return
		}

		next(w, r)
	}
}
