package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/ilyasaftr/ory-kratos-disposable/internal/domain"
	"github.com/ilyasaftr/ory-kratos-disposable/internal/httpx"
)

// ValidateHandler handles email validation requests from Ory Kratos
type ValidateHandler struct {
	disposableService EmailValidator
	logger            *slog.Logger
}

type EmailValidator interface {
	IsDisposable(email string) (bool, string, error)
}

type validateRequest struct {
	Email string `json:"email"`
}

const maxRequestBodyBytes = 1 << 20

// NewValidateHandler creates a new validation handler.
func NewValidateHandler(svc EmailValidator, log *slog.Logger) *ValidateHandler {
	return &ValidateHandler{
		disposableService: svc,
		logger:            log,
	}
}

// Handle processes the validation request
func (h *ValidateHandler) Handle(w http.ResponseWriter, r *http.Request) {
	// Use handler logger for all logging
	log := h.logger

	// Only accept POST requests
	if r.Method != http.MethodPost {
		h.respondError(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	req, err := h.decodeValidateRequest(w, r)
	if err != nil {
		log.Error("failed to decode request", slog.Any("error", err))
		h.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if req.Email == "" {
		h.respondError(w, http.StatusBadRequest, domain.ErrMissingEmail.Error())
		return
	}

	// Check if the email is disposable
	isDisposable, emailDomain, err := h.disposableService.IsDisposable(req.Email)
	if err != nil {
		log.Error("failed to check email",
			slog.Any("error", err),
			slog.String("email", req.Email))
		h.respondError(w, http.StatusBadRequest, "Invalid email format")
		return
	}

	// If disposable, return error response to interrupt the flow
	if isDisposable {
		log.Info("disposable email detected",
			slog.String("email", req.Email),
			slog.String("domain", emailDomain),
		)

		errorResp := domain.NewErrorResponse(req.Email, emailDomain)
		h.respondJSON(w, http.StatusBadRequest, errorResp)
		return
	}

	// Email is valid - allow flow to continue
	log.Info("email validated successfully",
		slog.String("email", req.Email))

	// Return 200 OK with empty response
	h.respondJSON(w, http.StatusOK, struct{}{})
}

func (h *ValidateHandler) decodeValidateRequest(w http.ResponseWriter, r *http.Request) (*validateRequest, error) {
	// Limit body size to prevent abuse.
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req validateRequest
	if err := dec.Decode(&req); err != nil {
		return nil, err
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return nil, err
	}

	return &req, nil
}

// respondError sends an error response
func (h *ValidateHandler) respondError(w http.ResponseWriter, statusCode int, message string) {
	if err := httpx.WriteOryError(w, statusCode, message); err != nil {
		h.logger.Error("failed to encode error response", slog.Any("error", err))
	}
}

// respondJSON sends a JSON response
func (h *ValidateHandler) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	if err := httpx.WriteJSON(w, statusCode, data); err != nil {
		h.logger.Error("failed to encode response", slog.Any("error", err))
	}
}
