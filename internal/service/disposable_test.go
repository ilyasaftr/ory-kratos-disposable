package service

import (
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ilyasaftr/ory-kratos-disposable/internal/domain"
)

func TestIsDisposable_DegradedModeFailOpen_Allows(t *testing.T) {
	svc := NewDisposableEmailService(
		[]string{"https://example.com/list.txt"},
		30*time.Minute,
		domain.FailureModeOpen,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	disposable, extractedDomain, err := svc.IsDisposable("user@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if disposable {
		t.Fatalf("expected fail-open mode to allow email")
	}
	if extractedDomain != "example.com" {
		t.Fatalf("expected extracted domain example.com, got %s", extractedDomain)
	}
}

func TestIsDisposable_DegradedModeFailClosed_Blocks(t *testing.T) {
	svc := NewDisposableEmailService(
		[]string{"https://example.com/list.txt"},
		30*time.Minute,
		domain.FailureModeClosed,
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)

	disposable, extractedDomain, err := svc.IsDisposable("user@example.com")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !disposable {
		t.Fatalf("expected fail-closed mode to block email")
	}
	if extractedDomain != "example.com" {
		t.Fatalf("expected extracted domain example.com, got %s", extractedDomain)
	}
}
