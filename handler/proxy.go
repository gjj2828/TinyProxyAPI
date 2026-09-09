// handler/proxy.go
package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/user/TinyProxyAPI/matcher"
)

type ProxyHandler struct {
	matcher *matcher.Matcher
}

func NewProxyHandler(m *matcher.Matcher) *ProxyHandler {
	return &ProxyHandler{matcher: m}
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

	// 匹配平台
	platform, ok := h.matcher.Match(reqBody.Model)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"message": "model not found",
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
	for _, key := range h.matcher.GetAllowedKeys() {
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