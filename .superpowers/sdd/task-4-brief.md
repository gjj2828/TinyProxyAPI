# Task 4: ProxyHandler 实现

## 概述

实现请求处理器，使用 httputil.ReverseProxy 实现透传。

## Global Constraints（必须遵守）

- Go 1.21+
- 监听地址：`:3178`
- 端点：`POST /v1/responses`
- 配置文件：`config.toml`（TOML 格式）
- 热重载：SIGHUP 信号
- 模型匹配：全量匹配

## 工作目录

`D:/Documents/Projects/TinyProxyAPI`

## 接口

```go
package handler

import (
    "encoding/json"
    "net/http"
    "net/http/httputil"
    "net/url"
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
```

## 具体任务

### Step 1: 编写 ProxyHandler 测试

```go
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
        Default: config.DefaultConfig{Platform: "test"},
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
```

### Step 2: 运行测试确认失败

Run: `go test ./handler/... -v`
Expected: FAIL - undefined functions/structs

### Step 3: 实现 ProxyHandler

实现 ServeHTTP 方法，包括：
- 仅处理 POST /v1/responses，其他返回 404
- 解析 body 获取 model 字段
- model 未匹配返回 404
- 解析 body 失败返回 400
- 使用 ReverseProxy 透传请求

### Step 4: 运行测试确认通过

Run: `go test ./handler/... -v`
Expected: PASS

### Step 5: 提交

```bash
git add handler/proxy.go handler/proxy_test.go
git commit -m "feat: add ProxyHandler with ReverseProxy passthrough"
```

## 验收标准

1. handler/proxy.go 存在，ServeHTTP 正确处理请求
2. 仅处理 POST /v1/responses，其他路径返回 404
3. model 未匹配时返回 404
4. body 解析失败时返回 400
5. handler/proxy_test.go 测试通过
6. 提交已创建
