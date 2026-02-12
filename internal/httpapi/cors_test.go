package httpapi

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"ble-printer-bridge/internal/config"
	"ble-printer-bridge/internal/logging"
)

func newTestLogger(t *testing.T) *logging.Logger {
	t.Helper()
	path := filepath.Join(t.TempDir(), "app.log")
	logger, err := logging.New(path, true)
	if err != nil {
		t.Fatalf("new logger: %v", err)
	}
	t.Cleanup(func() {
		logger.Close()
	})
	return logger
}

func TestCORSExplicitOriginAllowed(t *testing.T) {
	cfg := config.Config{}
	config.ApplyDefaults(&cfg)
	cfg.CORS.AllowOrigins = "https://allowed.example"
	logger := newTestLogger(t)
	cors := newCORSConfig(&cfg, logger)

	handler := corsMiddleware(logger, cors, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/health", nil)
	req.Header.Set("Origin", "https://allowed.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://allowed.example" {
		t.Fatalf("expected ACAO header, got %q", got)
	}
}

func TestCORSExplicitOriginAllowedWithTrailingSlashInConfig(t *testing.T) {
	cfg := config.Config{}
	config.ApplyDefaults(&cfg)
	cfg.CORS.AllowOrigins = "https://allowed.example/"
	logger := newTestLogger(t)
	cors := newCORSConfig(&cfg, logger)

	handler := corsMiddleware(logger, cors, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/health", nil)
	req.Header.Set("Origin", "https://allowed.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://allowed.example" {
		t.Fatalf("expected ACAO header, got %q", got)
	}
}

func TestCORSPatternOriginAllowed(t *testing.T) {
	cfg := config.Config{}
	config.ApplyDefaults(&cfg)
	cfg.CORS.AllowOriginPatterns = "https://integrated-pos-web-pr-*.onrender.com"
	logger := newTestLogger(t)
	cors := newCORSConfig(&cfg, logger)

	handler := corsMiddleware(logger, cors, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/health", nil)
	req.Header.Set("Origin", "https://integrated-pos-web-pr-131.onrender.com")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://integrated-pos-web-pr-131.onrender.com" {
		t.Fatalf("expected ACAO header, got %q", got)
	}
}

func TestCORSDisallowedOriginNoACAO(t *testing.T) {
	cfg := config.Config{}
	config.ApplyDefaults(&cfg)
	logger := newTestLogger(t)
	cors := newCORSConfig(&cfg, logger)

	handler := corsMiddleware(logger, cors, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1/health", nil)
	req.Header.Set("Origin", "https://not-allowed.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no ACAO header, got %q", got)
	}
}

func TestCORSPreflightOptions(t *testing.T) {
	cfg := config.Config{}
	config.ApplyDefaults(&cfg)
	cfg.CORS.AllowOrigins = "https://allowed.example"
	logger := newTestLogger(t)
	cors := newCORSConfig(&cfg, logger)

	handler := corsMiddleware(logger, cors, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "http://127.0.0.1/health", nil)
	req.Header.Set("Origin", "https://allowed.example")
	req.Header.Set("Access-Control-Request-Method", "GET")
	req.Header.Set("Access-Control-Request-Headers", "content-type,x-api-key")
	req.Header.Set("Access-Control-Request-Private-Network", "true")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204 got %d", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got == "" {
		t.Fatalf("expected Allow-Methods header")
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Fatalf("expected Allow-Headers header")
	}
	if got := rec.Header().Get("Access-Control-Allow-Private-Network"); got != "true" {
		t.Fatalf("expected Allow-Private-Network header, got %q", got)
	}
}

func TestCORSOptionsBypassAuth(t *testing.T) {
	cfg := config.Config{}
	config.ApplyDefaults(&cfg)
	cfg.Auth.ApiKey = "test-key"
	cfg.CORS.AllowOrigins = "https://allowed.example"
	logger := newTestLogger(t)
	cors := newCORSConfig(&cfg, logger)

	s := &Server{cfg: &cfg, log: logger}
	called := false
	protected := s.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	handler := corsMiddleware(logger, cors, http.HandlerFunc(protected))
	req := httptest.NewRequest(http.MethodOptions, "http://127.0.0.1/print/text", nil)
	req.Header.Set("Origin", "https://allowed.example")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected status 204 got %d", rec.Code)
	}
	if called {
		t.Fatalf("expected auth handler to be bypassed for OPTIONS")
	}
}
