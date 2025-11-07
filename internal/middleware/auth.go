package middleware

import (
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ilyasaftr/ory-kratos-disposable/internal/domain"
)

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
		apiKey := r.Header.Get("X-API-Key")

		if apiKey == "" {
			m.logger.Warn("missing API key",
				slog.String("path", r.URL.Path),
				slog.String("method", r.Method),
				slog.String("ip", r.RemoteAddr))
			respondError(w, http.StatusUnauthorized, "Missing API key")
			return
		}

		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(m.apiKey)) != 1 {
			m.logger.Warn("invalid API key",
				slog.String("path", r.URL.Path),
				slog.String("method", r.Method),
				slog.String("ip", r.RemoteAddr))
			respondError(w, http.StatusUnauthorized, "Invalid API key")
			return
		}

		next(w, r)
	}
}

// respondError sends a JSON error response
func respondError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	resp := domain.OryWebhookResponse{
		Messages: []domain.MessageGroup{
			{
				InstancePtr: "#/",
				Messages: []domain.Message{
					{
						ID:   statusCode,
						Text: message,
						Type: "error",
					},
				},
			},
		},
	}

	json.NewEncoder(w).Encode(resp)
}
