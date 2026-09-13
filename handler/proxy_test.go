// handler/proxy_test.go
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gjj2828/TinyProxyAPI/config"
	"github.com/gjj2828/TinyProxyAPI/matcher"
)

func TestProxyHandler_ServeHTTP(t *testing.T) {
	cfg := &config.Config{
		Default: config.DefaultConfig{Platform: "nonexistent"},
		AllowedKeys: []string{"valid-key-001", "valid-key-002"},
		Platforms: map[string]config.Platform{
			"test": {BaseURL: "https://api.example.com", APIKey: "sk-test-key"},
		},
		Models: map[string]string{
			"test-model": "test",
		},
	}
	m := matcher.NewMatcher(cfg)
	h := NewProxyHandler(m)

	// Test 404 for unknown model (fallback to nonexistent default platform)
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

	// Test 401 for missing Authorization header
	req = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"test-model"}`))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing auth header, got %d", w.Code)
	}

	// Test 401 for invalid API key
	req = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"test-model"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer invalid-key")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid key, got %d", w.Code)
	}
}

func TestProxyHandler_ListModels(t *testing.T) {
	cfg := &config.Config{
		AllowedKeys: []string{"valid-key-001"},
		Platforms: map[string]config.Platform{
			"openai":    {BaseURL: "https://api.openai.com", APIKey: "sk-test"},
			"anthropic": {BaseURL: "https://api.anthropic.com", APIKey: "sk-test"},
		},
		Models: map[string]string{
			"gpt-4o":                 "openai",
			"gpt-4o-mini":            "openai",
			"claude-sonnet-4-20250514": "anthropic",
		},
	}
	m := matcher.NewMatcher(cfg)
	h := NewProxyHandler(m)

	// Test 401 for missing Authorization header
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing auth, got %d", w.Code)
	}

	// Test 401 for invalid API key
	req = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer invalid-key")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid key, got %d", w.Code)
	}

	// Test 200 with valid key
	req = httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	req.Header.Set("Authorization", "Bearer valid-key-001")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	// Verify response body structure
	var resp struct {
		Object string `json:"object"`
		Data   []struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Object != "list" {
		t.Errorf("expected object='list', got '%s'", resp.Object)
	}
	if len(resp.Data) != 3 {
		t.Errorf("expected 3 models, got %d", len(resp.Data))
	}
	// Verify first model by looking it up by ID (avoid relying on map iteration order)
	var gpt4o *struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}
	for i := range resp.Data {
		if resp.Data[i].ID == "gpt-4o" {
			gpt4o = &resp.Data[i]
			break
		}
	}
	if gpt4o == nil {
		t.Fatalf("expected model 'gpt-4o' in response, got %+v", resp.Data)
	}
	if gpt4o.OwnedBy != "openai" {
		t.Errorf("expected gpt-4o owned_by='openai', got '%s'", gpt4o.OwnedBy)
	}
	if gpt4o.Created != 1788944949 {
		t.Errorf("expected gpt-4o created=1788944949, got %d", gpt4o.Created)
	}
}