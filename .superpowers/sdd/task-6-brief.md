# Task 6: Main 入口整合

## 概述

实现 main.go 主入口，整合所有组件并支持 SIGHUP 热重载。

## Global Constraints（必须遵守）

- Go 1.21+
- 监听地址：`:3178`
- 端点：`POST /v1/responses`
- 配置文件：`config.toml`（TOML 格式）
- 热重载：SIGHUP 信号
- 模型匹配：全量匹配

## 工作目录

`D:/Documents/Projects/TinyProxyAPI`

## 实现

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

## 具体任务

### Step 1: 创建 main.go

实现：
- 命令行参数 `-config` 指定配置文件路径
- 启动时加载配置
- 创建 Matcher 和 ProxyHandler
- 启动 HTTP 服务器
- 监听 SIGHUP 实现热重载

### Step 2: 创建 config.toml

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

### Step 3: 编译测试

Run: `go build -o TinyProxyAPI .`
Expected: 编译成功，生成 TinyProxyAPI 可执行文件

### Step 4: 提交

```bash
git add main.go config.toml
git commit -m "feat: add main entry point with hot reload support"
```

## 验收标准

1. main.go 存在，包含完整的 main 函数
2. config.toml 存在
3. 编译成功，生成 TinyProxyAPI 可执行文件
4. 提交已创建
