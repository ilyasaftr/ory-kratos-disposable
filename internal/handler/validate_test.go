package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubEmailValidator struct {
	isDisposable bool
	domain       string
	err          error
}

func (s stubEmailValidator) IsDisposable(email string) (bool, string, error) {
	return s.isDisposable, s.domain, s.err
}

func TestValidateHandle_RejectsTrailingPayload(t *testing.T) {
	h := NewValidateHandler(
		stubEmailValidator{isDisposable: false},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/validate/email", bytes.NewBufferString(`{"email":"user@example.com"}{"extra":true}`))
	rec := httptest.NewRecorder()

	h.Handle(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestValidateHandle_ReturnsBadRequestForDisposableEmail(t *testing.T) {
	h := NewValidateHandler(
		stubEmailValidator{isDisposable: true, domain: "tempmail.com"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/validate/email", bytes.NewBufferString(`{"email":"user@tempmail.com"}`))
	rec := httptest.NewRecorder()

	h.Handle(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestValidateHandle_ReturnsOKForValidEmail(t *testing.T) {
	h := NewValidateHandler(
		stubEmailValidator{isDisposable: false, domain: "example.com"},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	req := httptest.NewRequest(http.MethodPost, "/v1/validate/email", bytes.NewBufferString(`{"email":"user@example.com"}`))
	rec := httptest.NewRecorder()

	h.Handle(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected json response, got error: %v", err)
	}
}
