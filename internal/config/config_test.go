package config

import "testing"

func TestLoad_RejectsNonPositiveRefreshInterval(t *testing.T) {
	t.Setenv("WEBHOOK_API_KEY", "test-key")
	t.Setenv("DISPOSABLE_LIST_UPDATE_INTERVAL", "0s")
	t.Setenv("DISPOSABLE_FAILURE_MODE", "open")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error for non-positive refresh interval")
	}
}

func TestLoad_RejectsInvalidFailureMode(t *testing.T) {
	t.Setenv("WEBHOOK_API_KEY", "test-key")
	t.Setenv("DISPOSABLE_LIST_UPDATE_INTERVAL", "30m")
	t.Setenv("DISPOSABLE_FAILURE_MODE", "invalid")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected error for invalid failure mode")
	}
}

func TestLoad_NormalizesFailureMode(t *testing.T) {
	t.Setenv("WEBHOOK_API_KEY", "test-key")
	t.Setenv("DISPOSABLE_LIST_UPDATE_INTERVAL", "30m")
	t.Setenv("DISPOSABLE_FAILURE_MODE", " CLOSED ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected config load success, got error: %v", err)
	}
	if got, want := cfg.Failure.Mode.String(), "closed"; got != want {
		t.Fatalf("expected normalized mode %q, got %q", want, got)
	}
}
