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
		Default:     config.DefaultConfig{Platform: "nonexistent"},
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
			"gpt-4o":                   "openai",
			"gpt-4o-mini":              "openai",
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

// TestProxyHandler_ProtocolRouting 验证各协议端点透传到上游同名路径，
// 并按上游协议设置正确的鉴权头（且不泄露客户端凭证）。
func TestProxyHandler_ProtocolRouting(t *testing.T) {
	type captured struct {
		path             string
		authorization    string
		apiKey           string
		anthropicVersion string
	}
	capturedCh := make(chan captured, 1)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedCh <- captured{
			path:             r.URL.Path,
			authorization:    r.Header.Get("Authorization"),
			apiKey:           r.Header.Get("x-api-key"),
			anthropicVersion: r.Header.Get("anthropic-version"),
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	cfg := &config.Config{
		AllowedKeys: []string{"client-key"},
		Platforms: map[string]config.Platform{
			"up": {BaseURL: upstream.URL, APIKey: "sk-upstream"},
		},
		Models: map[string]string{
			"model-a": "up",
		},
	}
	h := NewProxyHandler(matcher.NewMatcher(cfg))

	cases := []struct {
		name      string
		path      string
		authHdr   string
		apiKeyHdr string
	}{
		{name: "responses", path: "/v1/responses", authHdr: "Bearer client-key"},
		{name: "openai", path: "/v1/chat/completions", authHdr: "Bearer client-key"},
		{name: "anthropic", path: "/v1/messages", apiKeyHdr: "client-key"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tc.path, bytes.NewBufferString(`{"model":"model-a"}`))
			req.Header.Set("Content-Type", "application/json")
			if tc.authHdr != "" {
				req.Header.Set("Authorization", tc.authHdr)
			}
			if tc.apiKeyHdr != "" {
				req.Header.Set("x-api-key", tc.apiKeyHdr)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d (%s)", w.Code, w.Body.String())
			}

			got := <-capturedCh
			if got.path != tc.path {
				t.Errorf("expected upstream path %s, got %s", tc.path, got.path)
			}

			if tc.name == "anthropic" {
				if got.apiKey != "sk-upstream" {
					t.Errorf("expected upstream x-api-key sk-upstream, got %q", got.apiKey)
				}
				if got.authorization != "" {
					t.Errorf("expected no Authorization forwarded, got %q", got.authorization)
				}
				if got.anthropicVersion == "" {
					t.Errorf("expected default anthropic-version to be set")
				}
			} else {
				if got.authorization != "Bearer sk-upstream" {
					t.Errorf("expected upstream Authorization 'Bearer sk-upstream', got %q", got.authorization)
				}
				if got.apiKey != "" {
					t.Errorf("expected no x-api-key forwarded, got %q", got.apiKey)
				}
			}
		})
	}
}

// TestProxyHandler_EndpointRouting 验证端点分发：不支持的端点返回 404，
// 新端点上的鉴权失败返回 401、未知模型返回 404。
func TestProxyHandler_EndpointRouting(t *testing.T) {
	cfg := &config.Config{
		AllowedKeys: []string{"client-key"},
		Platforms: map[string]config.Platform{
			"up": {BaseURL: "https://api.example.com", APIKey: "sk-upstream"},
		},
		Models: map[string]string{
			"model-a": "up",
		},
	}
	h := NewProxyHandler(matcher.NewMatcher(cfg))

	// 不支持的端点 → 404
	req := httptest.NewRequest(http.MethodPost, "/v1/embeddings", bytes.NewBufferString(`{"model":"model-a"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unsupported endpoint, got %d", w.Code)
	}

	// 新端点鉴权失败 → 401
	for _, path := range []string{"/v1/chat/completions", "/v1/messages"} {
		req = httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{"model":"model-a"}`))
		req.Header.Set("Authorization", "Bearer wrong-key")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for %s with wrong key, got %d", path, w.Code)
		}
	}

	// 新端点上未知模型 → 404
	req = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewBufferString(`{"model":"nope"}`))
	req.Header.Set("Authorization", "Bearer client-key")
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown model on chat endpoint, got %d", w.Code)
	}
}
