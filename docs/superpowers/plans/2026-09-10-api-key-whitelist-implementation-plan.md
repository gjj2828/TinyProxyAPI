# API Key 白名单验证实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 添加全局 API Key 白名单验证功能，验证客户端请求中 `Authorization: Bearer <key>` 是否在白名单中

**Architecture:** 在 ProxyHandler 中添加白名单验证逻辑，从 Matcher 获取白名单配置，按平台独立验证

**Tech Stack:** Go 1.21+, github.com/pelletier/go-toml/v2

## Global Constraints

- Go 1.21+
- 验证位置：handler/proxy.go 的 ServeHTTP 方法
- 白名单获取：matcher.GetAllowedKeys()
- 错误响应：401 Unauthorized + JSON error body

---

## File Structure

```
TinyProxyAPI/
├── config/
│   └── loader.go          # 修改：新增 AllowedKeys 字段
├── matcher/
│   └── matcher.go         # 修改：新增 GetAllowedKeys() 方法
├── handler/
│   └── proxy.go           # 修改：新增 API Key 验证逻辑
├── config.toml.example    # 修改：新增 allowed_keys 示例
└── *_test.go              # 各模块测试更新
```

---

### Task 1: ConfigLoader 支持 AllowedKeys

**Files:**
- Modify: `config/loader.go:10-15`
- Test: `config/loader_test.go`

**Interfaces:**
- Produces: `Config.AllowedKeys []string`

- [ ] **Step 1: 修改 Config 结构体，添加 AllowedKeys 字段**

```go
type Config struct {
    Server     ServerConfig
    Default    DefaultConfig
    AllowedKeys []string      // 新增
    Platforms  map[string]Platform
    Models     map[string]string
}
```

- [ ] **Step 2: 运行现有测试，确认不影响现有功能**

Run: `go test ./config/... -v`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add config/loader.go
git commit -m "feat: add AllowedKeys field to Config"
```

---

### Task 2: Matcher 添加 GetAllowedKeys 方法

**Files:**
- Modify: `matcher/matcher.go`
- Test: `matcher/matcher_test.go`

**Interfaces:**
- Consumes: `*config.Config`
- Produces: `GetAllowedKeys() []string`

- [ ] **Step 1: 在 matcher.go 末尾添加 GetAllowedKeys 方法**

```go
// GetAllowedKeys 返回全局 API Key 白名单
func (m *Matcher) GetAllowedKeys() []string {
    return m.cfg.AllowedKeys
}
```

- [ ] **Step 2: 运行现有测试，确认不影响现有功能**

Run: `go test ./matcher/... -v`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add matcher/matcher.go
git commit -m "feat: add GetAllowedKeys method to Matcher"
```

---

### Task 3: ProxyHandler 添加 API Key 验证

**Files:**
- Modify: `handler/proxy.go`
- Test: `handler/proxy_test.go`

**Interfaces:**
- Consumes: `Authorization` header, `h.matcher.GetAllowedKeys()`
- Produces: 401 错误响应或继续执行

- [ ] **Step 1: 在 ServeHTTP 方法中，解析 model 之后、转发请求之前，添加验证逻辑**

在 `// 匹配平台` 之前插入：

```go
// 验证 API Key 白名单
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

isValid := false
for _, key := range h.matcher.GetAllowedKeys() {
    if authHeader == "Bearer "+key {
        isValid = true
        break
    }
}
if !isValid {
    w.WriteHeader(http.StatusUnauthorized)
    json.NewEncoder(w).Encode(map[string]interface{}{
        "error": map[string]string{
            "message": "invalid api key",
            "type":    "invalid_request_error",
        },
    })
    return
}
```

- [ ] **Step 2: 运行测试**

Run: `go test ./handler/... -v`
Expected: PASS（现有测试应继续通过）

- [ ] **Step 3: 提交**

```bash
git add handler/proxy.go
git commit -m "feat: add API key whitelist validation in ProxyHandler"
```

---

### Task 4: 单元测试更新

**Files:**
- Modify: `config/loader_test.go`, `matcher/matcher_test.go`, `handler/proxy_test.go`
- Modify: `config.toml.example`

- [ ] **Step 1: 更新 config/loader_test.go，测试 AllowedKeys 解析**

在 TestLoadConfig 中添加：

```go
if len(cfg.AllowedKeys) != 2 || cfg.AllowedKeys[0] != "sk-key1" {
    t.Errorf("expected AllowedKeys [sk-key1, sk-key2], got %v", cfg.AllowedKeys)
}
```

更新测试配置文件内容：

```go
content := `
[server]
addr = ":3178"

[default]
platform = "openai"

[allowed_keys]
keys = ["sk-key1", "sk-key2"]

[platforms]
"openai" = { base_url = "https://api.openai.com", api_key = "sk-test" }

[models]
"gpt-4o" = "openai"
`
```

- [ ] **Step 2: 更新 matcher/matcher_test.go，测试 GetAllowedKeys**

添加测试：

```go
func TestMatcher_GetAllowedKeys(t *testing.T) {
    cfg := &config.Config{
        AllowedKeys: []string{"sk-key1", "sk-key2"},
    }
    m := NewMatcher(cfg)
    keys := m.GetAllowedKeys()
    if len(keys) != 2 || keys[0] != "sk-key1" {
        t.Errorf("expected [sk-key1, sk-key2], got %v", keys)
    }
}
```

- [ ] **Step 3: 更新 handler/proxy_test.go，测试 401 场景**

添加测试：

```go
// Test missing auth header
req := httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"test-model"}`))
req.Header.Set("Content-Type", "application/json")
w := httptest.NewRecorder()
h.ServeHTTP(w, req)
if w.Code != http.StatusUnauthorized {
    t.Errorf("expected 401 for missing auth, got %d", w.Code)
}

// Test invalid auth header
req = httptest.NewRequest(http.MethodPost, "/v1/responses", bytes.NewBufferString(`{"model":"test-model"}`))
req.Header.Set("Content-Type", "application/json")
req.Header.Set("Authorization", "Bearer invalid-key")
w = httptest.NewRecorder()
h.ServeHTTP(w, req)
if w.Code != http.StatusUnauthorized {
    t.Errorf("expected 401 for invalid key, got %d", w.Code)
}
```

- [ ] **Step 4: 更新 config.toml.example，添加 allowed_keys 示例**

```toml
[allowed_keys]
keys = ["sk-client-key-1", "sk-client-key-2"]
```

- [ ] **Step 5: 运行所有测试**

Run: `go test ./... -v`
Expected: ALL PASS

- [ ] **Step 6: 提交**

```bash
git add config/loader_test.go matcher/matcher_test.go handler/proxy_test.go config.toml.example
git commit -m "test: add tests for API key whitelist feature"
```

---

## Spec Coverage Check

| Spec Section | Task |
|--------------|------|
| 新增 `[allowed_keys]` 配置节 | Task 1, Task 4 |
| 验证 Authorization header | Task 3 |
| 不匹配时返回 401 | Task 3 |
| 单元测试 | Task 4 |

## Self-Review

- [x] 所有步骤包含实际代码
- [x] 测试代码完整
- [x] 类型一致性检查：Config.AllowedKeys → Matcher.GetAllowedKeys() → ProxyHandler 验证
- [x] 无 placeholder（TODO、TBD 等）
- [x] 每个 task 可独立测试
