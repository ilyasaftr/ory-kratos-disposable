package handler

import (
    "encoding/json"
    "log/slog"
    "net/http"

    "github.com/ilyasaftr/ory-kratos-disposable/internal/domain"
    "github.com/ilyasaftr/ory-kratos-disposable/internal/service"
)

// ValidateHandler handles email validation requests from Ory Kratos
type ValidateHandler struct {
    disposableService *service.DisposableEmailService
    logger            *slog.Logger
}

// NewValidateHandler creates a new validation handler
func NewValidateHandler(svc *service.DisposableEmailService, log *slog.Logger) *ValidateHandler {
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

    // Parse the request body (simplified payload: {"email":"..."})
    type ValidateRequest struct {
        Email string `json:"email"`
    }
    // Limit body size to prevent abuse
    r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB
    dec := json.NewDecoder(r.Body)
    dec.DisallowUnknownFields()

    var req ValidateRequest
    if err := dec.Decode(&req); err != nil {
        log.Error("failed to decode request", slog.Any("error", err))
        h.respondError(w, http.StatusBadRequest, "Invalid request body")
        return
    }
    if req.Email == "" {
        h.respondError(w, http.StatusBadRequest, "Email is required")
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

    // Return 200 OK to allow the flow to continue
    w.WriteHeader(http.StatusOK)
}

// respondError sends an error response
func (h *ValidateHandler) respondError(w http.ResponseWriter, statusCode int, message string) {
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
    h.respondJSON(w, statusCode, resp)
}

// respondJSON sends a JSON response
func (h *ValidateHandler) respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)

    if err := json.NewEncoder(w).Encode(data); err != nil {
        h.logger.Error("failed to encode response", slog.Any("error", err))
    }
}
