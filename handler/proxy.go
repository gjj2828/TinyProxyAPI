// handler/proxy.go
package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"

	"github.com/gjj2828/TinyProxyAPI/matcher"
)

// 支持的协议种类。
const (
	protocolResponses = "responses"
	protocolOpenAI    = "openai"
	protocolAnthropic = "anthropic"
)

// 客户端未携带 anthropic-version 时补的默认版本。
const defaultAnthropicVersion = "2023-06-01"

// endpointProtocol 把客户端请求路径映射到协议种类。
// 客户端调哪个端点，就以哪种协议透传到上游同名端点。
var endpointProtocol = map[string]string{
	"/v1/responses":        protocolResponses,
	"/v1/chat/completions": protocolOpenAI,
	"/v1/messages":         protocolAnthropic,
}

// ProxyHandler 是 HTTP server 持有的单例 handler。
// matcher 通过 atomic.Pointer 持有，支持运行时原子替换（热重载）。
type ProxyHandler struct {
	matcher atomic.Pointer[matcher.Matcher]
}

func NewProxyHandler(m *matcher.Matcher) *ProxyHandler {
	h := &ProxyHandler{}
	h.matcher.Store(m)
	return h
}

// Update 原子替换当前 matcher，用于配置热重载。
func (h *ProxyHandler) Update(m *matcher.Matcher) {
	h.matcher.Store(m)
}

// currentMatcher 返回当前 matcher。每个请求都重新 Load，
// 这样 Update 后所有后续请求立即生效。
func (h *ProxyHandler) currentMatcher() *matcher.Matcher {
	return h.matcher.Load()
}

func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Handle GET /v1/models
	if r.Method == http.MethodGet && r.URL.Path == "/v1/models" {
		h.handleListModels(w, r)
		return
	}

	// 按端点确定协议，仅处理已支持协议的 POST 请求
	protocol, supported := endpointProtocol[r.URL.Path]
	if r.Method != http.MethodPost || !supported {
		http.NotFound(w, r)
		return
	}

	// 读取 body 一次，后面需要复用
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"message": "failed to read request body",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	// 解析 body 获取 model
	var reqBody struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(bodyBytes, &reqBody); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"message": "invalid request body",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	// 恢复 body 给后续 ReverseProxy 使用
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// 匹配平台 —— 每次请求取最新 matcher，按 (协议, 模型) 路由
	m := h.currentMatcher()
	platform, err := m.Match(protocol, reqBody.Model)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		var msg string
		switch {
		case errors.Is(err, matcher.ErrModelNotConfigured):
			msg = "model not configured: " + reqBody.Model + " (add it to [models] or set [default].platform)"
		case errors.Is(err, matcher.ErrPlatformNotFound):
			msg = "default platform not found in [platforms]: " + m.GetDefaultPlatform()
		default:
			msg = "model not found: " + reqBody.Model
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"message": msg,
				"type":    "invalid_request_error",
			},
		})
		return
	}

	// API Key 验证（兼容 OpenAI 的 Authorization: Bearer 与 Anthropic 的 x-api-key）
	authKey := clientAPIKey(r)
	if authKey == "" {
		writeAuthError(w, "missing api key")
		return
	}
	if !isAllowedKey(m.GetAllowedKeys(), authKey) {
		writeAuthError(w, "invalid api key")
		return
	}

	// 构建目标 URL —— NewSingleHostReverseProxy 会把 r.URL.Path 追加到 target.Path 后面，
	// 所以这里只设置 Scheme/Host，不要手动设置 Path，否则会变成 /v1/responses/v1/responses。
	target, _ := url.Parse(platform.BaseURL)

	// 创建 ReverseProxy
	proxy := httputil.NewSingleHostReverseProxy(target)

	// 自定义 Director：替换 Host，并按上游协议设置鉴权头。
	// 客户端的凭证不会被转发：先清除，再写入平台自己的 key。
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		req.Header.Del("Authorization")
		req.Header.Del("x-api-key")
		if protocol == protocolAnthropic {
			req.Header.Set("x-api-key", platform.APIKey)
			if req.Header.Get("anthropic-version") == "" {
				req.Header.Set("anthropic-version", defaultAnthropicVersion)
			}
		} else {
			req.Header.Set("Authorization", "Bearer "+platform.APIKey)
		}
	}

	proxy.ServeHTTP(w, r)
}

// handleListModels 处理 GET /v1/models 请求，返回 OpenAI 兼容的模型列表
func (h *ProxyHandler) handleListModels(w http.ResponseWriter, r *http.Request) {
	m := h.currentMatcher()

	// API Key 验证（兼容 OpenAI 的 Authorization: Bearer 与 Anthropic 的 x-api-key）
	authKey := clientAPIKey(r)
	if authKey == "" {
		writeAuthError(w, "missing api key")
		return
	}
	if !isAllowedKey(m.GetAllowedKeys(), authKey) {
		writeAuthError(w, "invalid api key")
		return
	}

	// 构建模型列表
	models := m.GetModels()
	data := make([]struct {
		ID      string `json:"id"`
		Object  string `json:"object"`
		Created int64  `json:"created"`
		OwnedBy string `json:"owned_by"`
	}, 0, len(models))

	for modelID, platformID := range models {
		data = append(data, struct {
			ID      string `json:"id"`
			Object  string `json:"object"`
			Created int64  `json:"created"`
			OwnedBy string `json:"owned_by"`
		}{
			ID:      modelID,
			Object:  "model",
			Created: 1788944949,
			OwnedBy: platformID,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"object": "list",
		"data":   data,
	})
}

// clientAPIKey 提取客户端提交的 API Key。
// 兼容 OpenAI/Responses 风格的 Authorization: Bearer <key>
// 与 Anthropic 风格的 x-api-key: <key>。
func clientAPIKey(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return r.Header.Get("x-api-key")
}

// isAllowedKey 判断 key 是否在白名单中。
func isAllowedKey(keys []string, key string) bool {
	for _, k := range keys {
		if k == key {
			return true
		}
	}
	return false
}

// writeAuthError 输出 OpenAI 风格的 401 错误响应。
func writeAuthError(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]string{
			"message": message,
			"type":    "invalid_request_error",
		},
	})
}
