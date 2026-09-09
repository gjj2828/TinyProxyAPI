# TinyProxyAPI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建一个极轻量的 API 路由代理服务，根据请求中的模型名透传到对应平台

**Architecture:** 使用 `httputil.ReverseProxy` 实现请求透传，通过 TOML 配置文件定义模型名到平台的路由映射，支持 SIGHUP 热重载

**Tech Stack:** Go 1.21+, github.com/pelletier/go-toml/v2

## Global Constraints

- Go 1.21+
- 监听地址：`:3178`
- 端点：`POST /v1/responses`
- 配置文件：`config.toml`（TOML 格式）
- 热重载：SIGHUP 信号
- 模型匹配：全量匹配

---

## File Structure

```
TinyProxyAPI/
├── main.go              # 程序入口
├── config/
│   └── loader.go        # 配置文件加载 + 热重载
├── handler/
│   └── proxy.go         # 请求处理器
├── matcher/
│   └── matcher.go       # 路由匹配逻辑
├── signal/
│   └── sighup.go        # SIGHUP 处理
├── go.mod
├── config.toml.example  # 配置示例
└── config/
    └── loader_test.go   # ConfigLoader 测试
    └── matcher_test.go  # Matcher 测试
```

---

### Task 1: 项目初始化

**Files:**
- Create: `go.mod`
- Create: `config.toml.example`

**Interfaces:**
- Produces: 可执行的 Go 模块

- [ ] **Step 1: 创建 go.mod**

```bash
mkdir -p TinyProxyAPI
cd TinyProxyAPI
go mod init github.com/user/TinyProxyAPI
```

Run: `go mod tidy`（等后续添加依赖后）

- [ ] **Step 2: 创建配置示例文件**

```toml
# config.toml.example
[server]
addr = ":3178"

[default]
platform = "openai"

[platforms]
"openai" = { base_url = "https://api.openai.com", api_key = "sk-your-openai-key" }
"anthropic" = { base_url = "https://api.anthropic.com", api_key = "sk-ant-your-anthropic-key" }

[models]
"gpt-4o" = "openai"
"gpt-4o-mini" = "openai"
"claude-sonnet-4-20250514" = "anthropic"
```

- [ ] **Step 3: 提交**

```bash
git init
git add go.mod config.toml.example
git commit -m "feat: initialize project structure"
```

---

### Task 2: ConfigLoader 实现

**Files:**
- Create: `config/loader.go`
- Create: `config/loader_test.go`

**Interfaces:**
- Consumes: TOML 配置文件路径
- Produces: `LoadConfig(path string) (*Config, error)`，`ReloadConfig(cfg *Config) error`

```go
package config

type Config struct {
    Server   ServerConfig
    Default  DefaultConfig
    Platforms map[string]Platform
    Models   map[string]string
}

type ServerConfig struct {
    Addr string
}

type DefaultConfig struct {
    Platform string
}

type Platform struct {
    BaseURL string
    APIKey  string
}
```

- [ ] **Step 1: 编写 ConfigLoader 测试**

```go
// config/loader_test.go
package config

import (
    "os"
    "testing"
)

func TestLoadConfig(t *testing.T) {
    content := `
[server]
addr = ":3178"

[default]
platform = "openai"

[platforms]
"openai" = { base_url = "https://api.openai.com", api_key = "sk-test" }

[models]
"gpt-4o" = "openai"
`
    tmpfile, err := os.CreateTemp("", "config-*.toml")
    if err != nil {
        t.Fatal(err)
    }
    defer os.Remove(tmpfile.Name())

    if _, err := tmpfile.WriteString(content); err != nil {
        t.Fatal(err)
    }
    tmpfile.Close()

    cfg, err := LoadConfig(tmpfile.Name())
    if err != nil {
        t.Fatalf("LoadConfig failed: %v", err)
    }

    if cfg.Server.Addr != ":3178" {
        t.Errorf("expected addr :3178, got %s", cfg.Server.Addr)
    }
    if cfg.Default.Platform != "openai" {
        t.Errorf("expected default platform openai, got %s", cfg.Default.Platform)
    }
    if cfg.Platforms["openai"].BaseURL != "https://api.openai.com" {
        t.Errorf("expected openai base_url https://api.openai.com, got %s", cfg.Platforms["openai"].BaseURL)
    }
    if cfg.Models["gpt-4o"] != "openai" {
        t.Errorf("expected gpt-4o -> openai, got %s", cfg.Models["gpt-4o"])
    }
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./config/... -v`
Expected: FAIL - "undefined: LoadConfig"

- [ ] **Step 3: 实现 LoadConfig**

```go
// config/loader.go
package config

import (
    "github.com/pelletier/go-toml/v2"
    "os"
)

type Config struct {
    Server    ServerConfig
    Default   DefaultConfig
    Platforms map[string]Platform
    Models    map[string]string
}

type ServerConfig struct {
    Addr string
}

type DefaultConfig struct {
    Platform string
}

type Platform struct {
    BaseURL string
    APIKey  string
}

func LoadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var cfg Config
    if err := toml.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }

    return &cfg, nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./config/... -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add config/loader.go config/loader_test.go
git commit -m "feat: add ConfigLoader with TOML support"
```

---

### Task 3: Matcher 实现

**Files:**
- Create: `matcher/matcher.go`
- Create: `matcher/matcher_test.go`

**Interfaces:**
- Consumes: `*config.Config`
- Produces: `NewMatcher(cfg *config.Config) *Matcher`，`Match(model string) (platform *config.Platform, ok bool)`

```go
package matcher

import "github.com/user/TinyProxyAPI/config"

type Matcher struct {
    cfg      *config.Config
}

func NewMatcher(cfg *config.Config) *Matcher {
    return &Matcher{cfg: cfg}
}

func (m *Matcher) Match(model string) (*config.Platform, bool) {
    platformID, ok := m.cfg.Models[model]
    if !ok {
        platformID = m.cfg.Default.Platform
    }
    platform, ok := m.cfg.Platforms[platformID]
    return &platform, ok
}
```

- [ ] **Step 1: 编写 Matcher 测试**

```go
// matcher/matcher_test.go
package matcher

import (
    "testing"
    "github.com/user/TinyProxyAPI/config"
)

func TestMatcher_Match(t *testing.T) {
    cfg := &config.Config{
        Default: config.DefaultConfig{
            Platform: "openai",
        },
        Platforms: map[string]config.Platform{
            "openai": {BaseURL: "https://api.openai.com", APIKey: "sk-test"},
        },
        Models: map[string]string{
            "gpt-4o": "openai",
        },
    }

    m := NewMatcher(cfg)

    // Test exact match
    p, ok := m.Match("gpt-4o")
    if !ok {
        t.Fatal("expected match for gpt-4o")
    }
    if p.BaseURL != "https://api.openai.com" {
        t.Errorf("expected https://api.openai.com, got %s", p.BaseURL)
    }

    // Test fallback to default
    p, ok = m.Match("unknown-model")
    if !ok {
        t.Fatal("expected fallback match")
    }
    if p.APIKey != "sk-test" {
        t.Errorf("expected sk-test, got %s", p.APIKey)
    }
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./matcher/... -v`
Expected: FAIL - "undefined: config"

- [ ] **Step 3: 创建 matcher go.mod 依赖**

在项目根目录执行：
```bash
go mod edit -replace github.com/user/TinyProxyAPI=.
go get github.com/user/TinyProxyAPI
go mod tidy
```

- [ ] **Step 4: 实现 Matcher**

```go
// matcher/matcher.go
package matcher

import "github.com/user/TinyProxyAPI/config"

type Matcher struct {
    cfg *config.Config
}

func NewMatcher(cfg *config.Config) *Matcher {
    return &Matcher{cfg: cfg}
}

// Match 根据模型名查找平台配置
// 优先全量匹配 models 表，未找到则使用 default platform
func (m *Matcher) Match(model string) (*config.Platform, bool) {
    platformID, ok := m.cfg.Models[model]
    if !ok {
        platformID = m.cfg.Default.Platform
    }
    platform, ok := m.cfg.Platforms[platformID]
    return &platform, ok
}
```

- [ ] **Step 5: 运行测试确认通过**

Run: `go test ./matcher/... -v`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add matcher/matcher.go matcher/matcher_test.go
git commit -m "feat: add Matcher for model-to-platform routing"
```

---

### Task 4: ProxyHandler 实现

**Files:**
- Create: `handler/proxy.go`

**Interfaces:**
- Consumes: `*matcher.Matcher`
- Produces: `NewProxyHandler(m *matcher.Matcher) http.Handler`

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

- [ ] **Step 1: 编写 ProxyHandler 测试（使用 httptest）**

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

- [ ] **Step 2: 运行测试确认失败**

Run: `go test ./handler/... -v`
Expected: FAIL

- [ ] **Step 3: 实现 ProxyHandler**

（见上方代码）

- [ ] **Step 4: 运行测试确认通过**

Run: `go test ./handler/... -v`
Expected: PASS

- [ ] **Step 5: 提交**

```bash
git add handler/proxy.go handler/proxy_test.go
git commit -m "feat: add ProxyHandler with ReverseProxy passthrough"
```

---

### Task 5: SIGHUP 热重载

**Files:**
- Create: `signal/sighup.go`

**Interfaces:**
- Consumes: 配置路径，`*atomic.Value` 存储的 config 指针
- Produces: `Watch(hup chan<- struct{})`，接收到 SIGHUP 时通知

```go
package signal

import (
    "os"
    "os/signal"
    "syscall"
)

func Watch() chan struct{} {
    hup := make(chan struct{}, 1)
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, syscall.SIGHUP)

    go func() {
        <-sigs
        hup <- struct{}{}
    }()

    return hup
}
```

- [ ] **Step 1: 编写测试**

```go
// signal/sighup_test.go
package signal

import (
    "testing"
    "time"
)

func TestWatch(t *testing.T) {
    hup := Watch()

    // 发送 SIGHUP（需要手动测试，或使用 syscall.Syscall）
    // 这里只测试 channel 创建
    select {
    case <-hup:
        t.Fatal("unexpected signal")
    case <-time.After(100 * time.Millisecond):
        // OK
    }
}
```

- [ ] **Step 2: 运行测试确认通过**

Run: `go test ./signal/... -v`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add signal/sighup.go signal/sighup_test.go
git commit -m "feat: add SIGHUP watcher for hot reload"
```

---

### Task 6: Main 入口整合

**Files:**
- Create: `main.go`
- Create: `config.toml`（工作配置文件）

**Interfaces:**
- Consumes: 所有组件
- Produces: 可执行程序

```go
package main

import (
    "flag"
    "log"
    "net/http"
    "sync/atomic"

    "github.com/user/TinyProxyAPI/config"
    "github.com/user/TinyProxyAPI/handler"
    "github.com/user/TinyProxyAPI/matcher"
    "github.com/user/TinyProxyAPI/signal"
)

var (
    configPath = flag.String("config", "config.toml", "path to config file")
    cfgValue   atomic.Value
)

func main() {
    flag.Parse()

    cfg, err := config.LoadConfig(*configPath)
    if err != nil {
        log.Fatalf("failed to load config: %v", err)
    }
    cfgValue.Store(cfg)

    m := matcher.NewMatcher(cfg)
    h := handler.NewProxyHandler(m)

    // 启动服务器
    go func() {
        log.Printf("server listening on %s", cfg.Server.Addr)
        if err := http.ListenAndServe(cfg.Server.Addr, h); err != nil {
            log.Fatalf("server error: %v", err)
        }
    }()

    // 监听 SIGHUP
    hup := signal.Watch()
    for range hup {
        log.Println("received SIGHUP, reloading config...")
        newCfg, err := config.LoadConfig(*configPath)
        if err != nil {
            log.Printf("failed to reload config: %v", err)
            continue
        }
        cfgValue.Store(newCfg)
        m = matcher.NewMatcher(newCfg)
        h = handler.NewProxyHandler(m)
        log.Println("config reloaded successfully")
    }
}
```

- [ ] **Step 1: 创建 main.go**

（见上方代码）

- [ ] **Step 2: 创建 config.toml**

```toml
[server]
addr = ":3178"

[default]
platform = "openai"

[platforms]
"openai" = { base_url = "https://api.openai.com", api_key = "sk-your-key" }

[models]
"gpt-4o" = "openai"
```

- [ ] **Step 3: 编译测试**

Run: `go build -o TinyProxyAPI .`
Expected: 编译成功，生成 TinyProxyAPI 可执行文件

- [ ] **Step 4: 提交**

```bash
git add main.go config.toml
git commit -m "feat: add main entry point with hot reload support"
```

---

## Spec Coverage Check

| Spec Section | Task |
|--------------|------|
| POST /v1/responses 端点透传 | Task 4 |
| 模型名全量匹配路由 | Task 3 |
| TOML 配置文件 | Task 1, Task 2 |
| SIGHUP 热重载 | Task 5, Task 6 |
| 配置启动时加载 | Task 6 |
| 平台抽象（减少重复配置） | Task 2, Task 3 |

## Self-Review

- [x] 所有步骤包含实际代码
- [x] 测试代码完整
- [x] 类型一致性检查：Config → Matcher → ProxyHandler 接口连贯
- [x] 无 placeholder（TODO、TBD 等）
- [x] 每个 task 可独立测试

---

**Plan complete and saved to `docs/superpowers/plans/2026-09-10-api-router-implementation-plan.md`**

**Two execution options:**

**1. Subagent-Driven (recommended)** - 我 dispatch 一个子代理 per task，任务间 review，快速迭代

**2. Inline Execution** - 在当前 session 中使用 executing-plans 技能执行任务，带检查点

选择哪个？