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
	"sync/atomic"

	"github.com/user/TinyProxyAPI/matcher"
)

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
	// 仅处理 POST /v1/responses
	if r.Method != http.MethodPost || r.URL.Path != "/v1/responses" {
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

	// 匹配平台 —— 每次请求取最新 matcher
	m := h.currentMatcher()
	platform, err := m.Match(reqBody.Model)
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

	// API Key 验证
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"message": "missing authorization header",
				"type":    "invalid_request_error",
			},
		})
		return
	}
	// 检查 Bearer <key> 是否在白名单中
	authKey := ""
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		authKey = authHeader[7:]
	}
	valid := false
	for _, key := range m.GetAllowedKeys() {
		if key == authKey {
			valid = true
			break
		}
	}
	if !valid {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"message": "invalid api key",
				"type":    "invalid_request_error",
			},
		})
		return
	}

	// 构建目标 URL
	target, _ := url.Parse(platform.BaseURL)
	target.Path = "/v1/responses"

	// 创建 ReverseProxy
	proxy := httputil.NewSingleHostReverseProxy(target)

	// 自定义 Director：替换 Host 头和 Authorization
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Host = target.Host
		req.Header.Set("Authorization", "Bearer "+platform.APIKey)
	}

	proxy.ServeHTTP(w, r)
}