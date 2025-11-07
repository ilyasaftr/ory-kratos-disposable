package service

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ilyasaftr/ory-kratos-disposable/internal/domain"
)

// DisposableEmailService manages the disposable email domain list
type DisposableEmailService struct {
	listURLs        []string
	refreshInterval time.Duration
	logger          *slog.Logger
	httpClient      *http.Client

	mu          sync.RWMutex
	domains     map[string]bool
	lastRefresh time.Time
	isReady     bool
	etags       map[string]string
}

func NewDisposableEmailService(listURLs []string, refreshInterval time.Duration, log *slog.Logger) *DisposableEmailService {
	return &DisposableEmailService{
		listURLs:        listURLs,
		refreshInterval: refreshInterval,
		logger:          log,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		domains: make(map[string]bool),
		etags:   make(map[string]string),
	}
}

// Start initializes the service and starts the auto-refresh goroutine
// The service always starts even if initial load fails (fail mode)
func (s *DisposableEmailService) Start(ctx context.Context) error {
	// Try initial load
	if err := s.refresh(); err != nil {
		// Always allow service to start in degraded mode
		s.logger.Warn("failed initial load - starting in FAIL mode (allowing all)",
			slog.Any("error", err),
			slog.Int("urls_tried", len(s.listURLs)))
		// isReady stays false, but service still starts
		go s.autoRefresh(ctx)
		return nil
	}

	// Success - start background refresh
	go s.autoRefresh(ctx)
	return nil
}

// autoRefresh periodically refreshes the disposable domains list
func (s *DisposableEmailService) autoRefresh(ctx context.Context) {
	ticker := time.NewTicker(s.refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("stopping auto-refresh goroutine")
			return
		case <-ticker.C:
			if err := s.refresh(); err != nil {
				s.logger.Error("failed to refresh disposable domains", slog.Any("error", err))
			}
		}
	}
}

// refresh fetches and updates the disposable domains list
// Tries all URLs in sequence until one succeeds
// On failure with existing data: keeps old data
// On failure without data: logs error for fail mode
func (s *DisposableEmailService) refresh() error {
	s.logger.Info("refreshing disposable domains list",
		slog.Int("urls", len(s.listURLs)))

	var lastErr error

	// Try each URL in sequence until one succeeds
	for i, url := range s.listURLs {
		s.logger.Info("fetching disposable domains",
			slog.String("url", url),
			slog.Int("attempt", i+1),
			slog.Int("total", len(s.listURLs)))

		domains, newETag, status, err := s.fetchFromURL(url)
		if err != nil {
			lastErr = err
			s.logger.Warn("failed to fetch from URL, trying next",
				slog.String("url", url),
				slog.Any("error", err))
			continue
		}

		if status == http.StatusNotModified {
			// Data not modified at this source
			s.mu.Lock()
			if s.isReady {
				// Consider refresh successful; update lastRefresh timestamp
				s.lastRefresh = time.Now()
				s.mu.Unlock()
				s.logger.Info("disposable domains list not modified",
					slog.String("source_url", url))
				return nil
			}
			s.mu.Unlock()
			// No data yet, try next URL that may have data
			s.logger.Warn("received 304 Not Modified but service has no data yet, trying next URL",
				slog.String("url", url))
			continue
		}

		// SUCCESS - Update cache atomically
		s.mu.Lock()
		s.domains = domains
		s.lastRefresh = time.Now()
		s.isReady = true
		if newETag != "" {
			s.etags[url] = newETag
		}
		s.mu.Unlock()

		s.logger.Info("disposable domains list refreshed successfully",
			slog.String("source_url", url),
			slog.Int("domains_count", len(domains)),
			slog.Time("last_refresh", s.lastRefresh))

		return nil
	}

	// All URLs failed
	s.handleAllRefreshFailures(lastErr)
	return fmt.Errorf("all %d URLs failed, last error: %w", len(s.listURLs), lastErr)
}

// fetchFromURL attempts to fetch and parse the domain list from a single URL
func (s *DisposableEmailService) fetchFromURL(url string) (map[string]bool, string, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to create request: %w", err)
	}

	// Add conditional request header if we have an ETag
	s.mu.RLock()
	etag := s.etags[url]
	s.mu.RUnlock()
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, "", 0, fmt.Errorf("failed to fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		// 304 Not Modified; return status for caller to handle
		return nil, "", http.StatusNotModified, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", resp.StatusCode, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Parse the TXT file (one domain per line)
	domains, err := s.parseTxtFile(resp.Body)
	if err != nil {
		return nil, "", resp.StatusCode, fmt.Errorf("failed to parse: %w", err)
	}

	// Capture ETag for future conditional requests
	newETag := resp.Header.Get("ETag")
	return domains, newETag, http.StatusOK, nil
}

// handleAllRefreshFailures logs appropriate messages when all URLs fail
func (s *DisposableEmailService) handleAllRefreshFailures(lastErr error) {
	s.mu.RLock()
	hasData := s.isReady
	domainCount := len(s.domains)
	lastRefresh := s.lastRefresh
	s.mu.RUnlock()

	if hasData {
		// Have old data - keep using it
		oldDuration := time.Since(lastRefresh)
		s.logger.Error("all disposable URLs failed - CONTINUING WITH OLD DATA",
			slog.Any("error", lastErr),
			slog.Int("urls_tried", len(s.listURLs)),
			slog.Int("old_domains_count", domainCount),
			slog.Duration("data_age", oldDuration),
			slog.Time("last_successful_refresh", lastRefresh))
	} else {
		// Never successfully loaded - degraded mode (always allowing)
		s.logger.Error("all disposable URLs failed - RUNNING IN DEGRADED MODE (allowing all)",
			slog.Any("error", lastErr),
			slog.Int("urls_tried", len(s.listURLs)))
	}
}

// parseTxtFile parses a TXT file with one domain per line
func (s *DisposableEmailService) parseTxtFile(r io.Reader) (map[string]bool, error) {
	domains := make(map[string]bool)
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Normalize to lowercase
		domain := strings.ToLower(line)
		domains[domain] = true
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to scan file: %w", err)
	}

	if len(domains) == 0 {
		return nil, fmt.Errorf("no domains found in the list")
	}

	return domains, nil
}

// IsDisposable checks if an email address uses a disposable domain
func (s *DisposableEmailService) IsDisposable(email string) (bool, string, error) {
	// Extract domain from email
	emailDomain := extractDomain(email)
	if emailDomain == "" {
		return false, "", domain.ErrInvalidEmail
	}

	// Check if the service is ready
	s.mu.RLock()
	ready := s.isReady
	s.mu.RUnlock()

	if !ready {
		// Never successfully loaded data - always fail (allow request)
		s.logger.Warn("service not ready - allowing request (fail mode)",
			slog.String("email", email),
			slog.String("domain", emailDomain))
		return false, emailDomain, nil // false = not disposable = ALLOW
	}

	// Normal operation with data (might be old, but that's OK)
	s.mu.RLock()
	isDisposable := s.domains[emailDomain]
	s.mu.RUnlock()

	return isDisposable, emailDomain, nil
}

// IsReady returns whether the service is ready to handle requests
func (s *DisposableEmailService) IsReady() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isReady
}

// extractDomain extracts the domain part from an email address
func extractDomain(email string) string {
	email = strings.TrimSpace(strings.ToLower(email))

	// Simple email validation
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}

	domain := parts[1]
	if domain == "" {
		return ""
	}

	return domain
}
