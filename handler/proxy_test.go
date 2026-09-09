// handler/proxy_test.go
package handler

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/user/TinyProxyAPI/config"
	"github.com/user/TinyProxyAPI/matcher"
)

func TestProxyHandler_ServeHTTP(t *testing.T) {
	cfg := &config.Config{
		Default: config.DefaultConfig{Platform: ""},
		Platforms: map[string]config.Platform{
			"test": {BaseURL: "https://api.example.com", APIKey: "sk-test-key"},
		},
		Models: map[string]string{
			"test-model": "test",
		},
	}
	m := matcher.NewMatcher(cfg)
	h := NewProxyHandler(m)

	// Test 404 for unknown model
	req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"unknown"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	// Test 400 for invalid body
	req = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{invalid}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}