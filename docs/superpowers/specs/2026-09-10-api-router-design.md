# TinyProxyAPI 设计文档

## 概述

一个极轻量的 API 路由代理服务，根据请求中的模型名将请求透传到对应的平台，专注于最小化对请求 Header 和 Body 的修改。

## 功能范围

### 第一版（v1）

- [x] `POST /v1/responses` 端点透传
- [x] 模型名全量匹配路由
- [x] TOML 配置文件
- [x] SIGHUP 热重载
- [x] 配置启动时加载

### 未来扩展

- 其他端点支持（如 `/v1/chat/completions`）
- 动态模型发现
- 指标监控

---

## 架构

```
                    ┌──────────────────────────────────────┐
                    │              TinyProxyAPI             │
                    │                                      │
POST /v1/responses  │  ┌─────────┐   ┌──────────────────┐  │
───────────────────▶│  │  Router │──▶│  ConfigLoader    │  │
                    │  └────┬────┘   └──────────────────┘  │
                    │       │                              │
                    │       ▼                              │
                    │  ┌─────────────┐                     │
                    │  │  Matcher    │  根据 model 查表     │
                    │  └──────┬──────┘                     │
                    │         │                            │
                    │         ▼                            │
                    │  ┌─────────────┐                     │
                    │  │ ReverseProxy│  透传请求           │
                    │  └──────┬──────┘                     │
                    └─────────┼────────────────────────────┘
                              │
                              ▼
                    ┌─────────────────┐
                    │   Target API    │
                    │   (OpenAI 等)   │
                    └─────────────────┘
```

---

## 配置文件

### config.toml

```toml
[server]
addr = ":3178"

# 默认平台配置（当模型名未匹配时使用）
[default]
platform = "openai"

# 平台定义（一次配置）
[platforms]
"openai" = { base_url = "https://api.openai.com", api_key = "sk-xxx" }
"anthropic" = { base_url = "https://api.anthropic.com", api_key = "sk-ant-xxx" }

# 模型名 → 平台映射
# 模型名必须全量匹配
[models]
"gpt-4o" = "openai"
"gpt-4o-mini" = "openai"
"claude-sonnet-4-20250514" = "anthropic"
```

---

## 核心组件

### 1. ConfigLoader

负责加载和解析 TOML 配置文件。

```go
type Config struct {
    Server struct {
        Addr string
    }
    Default struct {
        Platform string // 默认平台 ID
    }
    Platforms map[string]Platform // platform id -> platform config
    Models    map[string]string   // model name -> platform id
}

type Platform struct {
    BaseURL string
    APIKey  string
}
```

### 2. Matcher

根据请求 body 中的 `model` 字段进行路由匹配，采用两级查找：

- 第一步：在 `models` 表中查找模型名对应的平台 ID
- 第二步：在 `platforms` 表中查找平台 ID 对应的 base_url 和 api_key
- 匹配失败：返回 404 `{"error": {"message": "model not found", "type": "invalid_request_error"}}`

### 3. ProxyHandler

请求处理器，使用 `httputil.ReverseProxy` 实现透传。

关键行为：
- 读取请求 body，解析 `model` 字段
- 调用 Matcher 获取目标配置
- 修改请求的 `Host` 头（为目标 host）
- 转发请求，保持原始 header 和 body 不变
- 透传原始响应

### 4. HotReload

配置热重载机制：

- 监听 SIGHUP 信号
- 使用 `atomic.Value` 存储配置，实现无锁切换
- 新请求使用新配置，在途请求不受影响

---

## 数据流

### 请求流程

1. 客户端发送 `POST /v1/responses` 请求
2. `ProxyHandler` 读取 body，解析 JSON 获取 `model` 值
3. `Matcher` 先查 `models` 表得到平台 ID，再查 `platforms` 表得到 base_url 和 api_key
4. 构建目标 URL：`targetURL = base_url + "/v1/responses"`
5. 创建 `ReverseProxy`，设置 `Director` 替换 host
6. 执行代理

### 请求转换示例

**原始请求（客户端 → TinyProxyAPI）：**
```
POST /v1/responses
Host: localhost:3178
Content-Type: application/json
Authorization: Bearer user-key
Body: {"model": "gpt-4o", "input": "hello"}
```

**转发请求（TinyProxyAPI → OpenAI）：**
```
POST /v1/responses
Host: api.openai.com
Content-Type: application/json
Authorization: Bearer sk-xxx  # 替换为配置的 apikey
Body: {"model": "gpt-4o", "input": "hello"}  # 完全透传
```

---

## 错误处理

| 场景 | 响应状态码 | 响应体 |
|------|-----------|--------|
| model 未匹配 | 404 | `{"error": {"message": "model not found", "type": "invalid_request_error"}}` |
| 目标服务错误 | 透传原始状态码 | 透传原始 body |
| 解析 body 失败 | 400 | `{"error": {"message": "invalid request body", "type": "invalid_request_error"}}` |
| 配置加载失败 | 启动失败 | 日志输出错误 |

---

## 项目结构

```
TinyProxyAPI/
├── main.go              # 程序入口
├── config/
│   └── loader.go        # 配置文件加载
├── handler/
│   └── proxy.go         # 请求处理器
├── matcher/
│   └── matcher.go       # 路由匹配逻辑
├── signal/
│   └── sighup.go        # SIGHUP 处理
├── go.mod
└── config.toml          # 配置文件
```

---

## 依赖

- Go 1.21+
- 标准库 `net/http`, `net/http/httputil`, `os/signal`
- 第三方 TOML 解析：`github.com/BurntSushi/toml`（或 `github.com/pelletier/go-toml/v2`）

---

## 测试策略

- 单元测试：ConfigLoader、Matcher
- 集成测试：完整请求透传测试
