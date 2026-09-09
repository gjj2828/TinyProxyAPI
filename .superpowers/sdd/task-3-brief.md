# Task 3: Matcher 实现

## 概述

实现路由匹配器，根据模型名查找对应的平台配置。

## Global Constraints（必须遵守）

- Go 1.21+
- 监听地址：`:3178`
- 端点：`POST /v1/responses`
- 配置文件：`config.toml`（TOML 格式）
- 热重载：SIGHUP 信号
- 模型匹配：全量匹配

## 工作目录

`D:/Documents/Projects/TinyProxyAPI`

## 数据结构

```go
package matcher

import "github.com/user/TinyProxyAPI/config"

type Matcher struct {
    cfg *config.Config
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

## 具体任务

### Step 1: 编写 Matcher 测试

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

### Step 2: 运行测试确认失败

Run: `go test ./matcher/... -v`
Expected: FAIL - "undefined: config" or compilation error

### Step 3: 创建 matcher go.mod 依赖

```bash
go mod edit -replace github.com/user/TinyProxyAPI=.
go get github.com/user/TinyProxyAPI
go mod tidy
```

### Step 4: 实现 Matcher

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

### Step 5: 运行测试确认通过

Run: `go test ./matcher/... -v`
Expected: PASS

### Step 6: 提交

```bash
git add matcher/matcher.go matcher/matcher_test.go
git commit -m "feat: add Matcher for model-to-platform routing"
```

## 验收标准

1. matcher/matcher.go 存在，包含 Matcher 结构和 Match 方法
2. Match 方法正确实现两级查找逻辑
3. matcher/matcher_test.go 测试通过
4. 提交已创建
