// handler/proxy.go
package handler

import (
	"encoding/json"
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

	// 解析 body 获取 model
	var reqBody struct {
		Model string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]string{
				"message": "invalid request body",
				"type":    "invalid_request_error",
			},
		})
		return
	}

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