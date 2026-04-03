package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type HealthHandler struct {
	disposableService ReadinessChecker
	logger            *slog.Logger
}

type ReadinessChecker interface {
	IsReady() bool
}

func NewHealthHandler(svc ReadinessChecker, log *slog.Logger) *HealthHandler {
	return &HealthHandler{
		disposableService: svc,
		logger:            log,
	}
}

// HealthResponse uses a minimal format (application/health+json)
// with status values: "healthy" or "unhealthy".
type HealthResponse struct {
	Status string `json:"status"`
}

func (h *HealthHandler) Handle(w http.ResponseWriter, r *http.Request) {
	isReady := h.disposableService.IsReady()

	status := "fail"
	code := http.StatusServiceUnavailable
	if isReady {
		status = "pass"
		code = http.StatusOK
	}

	resp := HealthResponse{Status: status}

	w.Header().Set("Content-Type", "application/health+json")
	w.WriteHeader(code)

	if err := json.NewEncoder(w).Encode(resp); err != nil {
		h.logger.Error("failed to encode health response", slog.Any("error", err))
	}
}
