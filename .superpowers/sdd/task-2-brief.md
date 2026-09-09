# Task 2: ConfigLoader 实现

## 概述

实现配置文件加载器，支持 TOML 格式配置解析。

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
```

## 具体任务

### Step 1: 编写 ConfigLoader 测试

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

### Step 2: 运行测试确认失败

Run: `go test ./config/... -v`
Expected: FAIL - "undefined: LoadConfig"

### Step 3: 实现 LoadConfig

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

### Step 4: 运行测试确认通过

Run: `go test ./config/... -v`
Expected: PASS

### Step 5: 提交

```bash
git add config/loader.go config/loader_test.go
git commit -m "feat: add ConfigLoader with TOML support"
```

## 验收标准

1. config/loader.go 存在，包含 Config、Platform 等数据结构
2. LoadConfig 函数正确解析 TOML 文件
3. config/loader_test.go 测试通过
4. 提交已创建
