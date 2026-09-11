# Models List 接口实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 新增 `GET /v1/models` 接口，以 OpenAI 兼容格式返回模型列表

**Architecture:** 在 `handler/proxy.go` 的 `ServeHTTP` 中新增 `GET /v1/models` 分支，复用现有鉴权逻辑，模型列表从 `currentMatcher().cfg.Models` 聚合

**Tech Stack:** Go, net/http, encoding/json

## Global Constraints

- 响应格式必须与 OpenAI `/v1/models` 兼容
- `created` 时间戳固定为 `1788944949`（2026-09-09 09:09:09 UTC）
- `owned_by` 字段填配置 `[models]` 的 value（平台名）
- 所有请求必须经过 `allowed_keys` 白名单鉴权

---

## Task 1: 添加 models 接口测试

**Files:**
- Modify: `handler/proxy_test.go`

**Interfaces:**
- Consumes: `NewProxyHandler`, `ServeHTTP`
- Produces: 新增测试函数 `TestProxyHandler_ListModels`

- [ ] **Step 1: 添加 models 列表测试**

在 `handler/proxy_test.go` 末尾添加：

```go
func TestProxyHandler_ListModels(t *testing.T) {
	cfg := &config.Config{
		AllowedKeys: []string{"valid-key-001"},
		Platforms: map[string]config.Platform{
			"openai":    {BaseURL: "https://api.openai.com", APIKey: "sk-test"},
			"anthropic": {BaseURL: "https://api.anthropic.com", APIKey: "sk-test"},
		},
		Models: map[string]string{
			"gpt-4o":                "openai",
			"gpt-4o-mini":           "openai",
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
	// Verify first model
	if resp.Data[0].ID != "gpt-4o" || resp.Data[0].OwnedBy != "openai" {
		t.Errorf("unexpected first model: %+v", resp.Data[0])
	}
	if resp.Data[0].Created != 1788944949 {
		t.Errorf("expected created=1788944949, got %d", resp.Data[0].Created)
	}
}
```

注意：需要添加 `encoding/json` import。

- [ ] **Step 2: 运行测试验证失败**

Run: `cd D:/Documents/Projects/TinyProxyAPI && go test ./handler/... -run TestProxyHandler_ListModels -v`
Expected: FAIL（`ServeHTTP` 尚不支持 `/v1/models` 路径）

- [ ] **Step 3: Commit**

```bash
git add handler/proxy_test.go
git commit -m "test: add models list endpoint tests"
```

---

## Task 2: 实现 models 接口

**Files:**
- Modify: `handler/proxy.go`

**Interfaces:**
- Consumes: `currentMatcher()`, `matcher.Matcher`
- Produces: `GET /v1/models` 端点

- [ ] **Step 1: 修改 ServeHTTP 添加 models 分支**

在 `handler/proxy.go` 的 `ServeHTTP` 方法中，将：

```go
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // 仅处理 POST /v1/responses
    if r.Method != http.MethodPost || r.URL.Path != "/v1/responses" {
        http.NotFound(w, r)
        return
    }
```

改为：

```go
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // GET /v1/models — 返回模型列表
    if r.Method == http.MethodGet && r.URL.Path == "/v1/models" {
        h.listModels(w, r)
        return
    }

    // 仅处理 POST /v1/responses
    if r.Method != http.MethodPost || r.URL.Path != "/v1/responses" {
        http.NotFound(w, r)
        return
    }
```

- [ ] **Step 2: 添加 listModels 方法**

在 `handler/proxy.go` 文件末尾（`proxy.go:151` 之后）添加：

```go
// listModels 处理 GET /v1/models 请求，返回 OpenAI 兼容的模型列表。
func (h *ProxyHandler) listModels(w http.ResponseWriter, r *http.Request) {
    // 鉴权
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
    authKey := ""
    if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
        authKey = authHeader[7:]
    }
    valid := false
    for _, key := range h.currentMatcher().GetAllowedKeys() {
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

    // 构建模型列表
    const createdAt = int64(1788944949)
    m := h.currentMatcher()
    models := make([]struct {
        ID      string `json:"id"`
        Object  string `json:"object"`
        Created int64  `json:"created"`
        OwnedBy string `json:"owned_by"`
    }, 0, len(m.GetModelMap()))
    for id, platform := range m.GetModelMap() {
        models = append(models, struct {
            ID      string `json:"id"`
            Object  string `json:"object"`
            Created int64  `json:"created"`
            OwnedBy string `json:"owned_by"`
        }{
            ID:      id,
            Object:  "model",
            Created: createdAt,
            OwnedBy: platform,
        })
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]interface{}{
        "object": "list",
        "data":   models,
    })
}
```

- [ ] **Step 3: 在 matcher.go 中添加 GetModelMap 方法**

`matcher.Matcher` 需要暴露 `GetModelMap()` 方法返回 `map[string]string`（模型名→平台名），这样 `listModels` 才能遍历生成列表。

在 `matcher/matcher.go` 末尾添加：

```go
// GetModelMap 返回所有模型的映射表：模型名 → 平台名
func (m *Matcher) GetModelMap() map[string]string {
    return m.cfg.Models
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `cd D:/Documents/Projects/TinyProxyAPI && go test ./handler/... -run TestProxyHandler_ListModels -v`
Expected: PASS

- [ ] **Step 5: 运行全部测试确保无回归**

Run: `cd D:/Documents/Projects/TinyProxyAPI && go test ./... -v`
Expected: 全部 PASS

- [ ] **Step 6: Commit**

```bash
git add handler/proxy.go matcher/matcher.go
git commit -m "feat: add GET /v1/models endpoint with OpenAI-compatible format"
```

---

## Task 3: 更新文档（如需要）

**Files:**
- Modify: `README.md`（如需添加接口说明）

此任务为可选项——如果现有 README 不需要更新则跳过。

- [ ] **Step 1: 检查 README 是否需要更新**

检查 `README.md` 中的"请求示例"或"接口说明"部分是否需要新增 `/v1/models` 示例。如需更新，添加：

```markdown
### 获取模型列表

```bash
curl http://localhost:3178/v1/models \
  -H "Authorization: Bearer your-client-key"
```
```

- [ ] **Step 2: Commit（如有变更）**

```bash
git add README.md
git commit -m "docs: add /v1/models endpoint example"
```
