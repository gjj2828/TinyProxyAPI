# TinyProxyAPI

极轻量的 API 路由代理服务，根据请求中的模型名将请求透传到对应的平台。

## 特性

- **透传** - 使用 `httputil.ReverseProxy`，最小化 Header 和 Body 修改
- **灵活路由** - 模型名全量匹配，通过 TOML 配置
- **平台抽象** - 同一平台的多个模型共享 base_url 和 api_key
- **热重载** - 改完 `config.toml` 保存即生效（基于 fsnotify 跨平台监听）；Linux/Mac 也可发 SIGHUP 触发

## 快速开始

### 配置文件

```toml
# config.toml
allowed_keys = ["your-client-key-here"]

[server]
addr = ":3178"

[default]
platform = "openai"

[platforms]
"openai" = { base_url = "https://api.openai.com", api_key = "sk-your-key" }
"anthropic" = { base_url = "https://api.anthropic.com", api_key = "sk-ant-your-key" }

[models]
"gpt-4o" = "openai"
"gpt-4o-mini" = "openai"
"claude-sonnet-4-20250514" = "anthropic"
```

> ⚠️ **TOML 坑**：`allowed_keys` 是顶层 key，必须放在**所有 `[table]` 之前**。
> 如果写在 `[default]` 之后，它会被归入 `[default]` 表，但由于 `DefaultConfig` 结构体没有 `AllowedKeys` 字段，这个值会被**静默忽略**，导致所有客户端 key 都被白名单拒绝、代理返回 401 `invalid api key`。

### 运行

```bash
go build -o TinyProxyAPI.exe .
./TinyProxyAPI.exe -config config.toml
```

> Windows 上 `go build -o name .` 不会自动补 `.exe` 后缀，必须显式指定。

### 请求示例

```bash
curl -X POST http://localhost:3178/v1/responses \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-4o", "input": "Hello"}'
```

## 配置热重载

**保存 `config.toml` 后自动生效**——程序通过 fsnotify 监听配置文件，跨平台都支持（Windows / macOS / Linux）。

如果想手动触发（仅 Linux/Mac）：

```bash
kill -HUP $(pidof TinyProxyAPI)
```

## 项目结构

```
TinyProxyAPI/
├── main.go           # 程序入口
├── config/
│   └── loader.go     # 配置加载
├── handler/
│   └── proxy.go      # 请求处理
├── matcher/
│   └── matcher.go    # 路由匹配
├── signal/
│   ├── sighup.go     # SIGHUP 信号处理（Linux/Mac）
│   └── watcher.go    # fsnotify 文件监听（跨平台）
├── config.toml       # 配置文件
└── config.toml.example
```

## 依赖

- Go 1.22+
- github.com/pelletier/go-toml/v2
- github.com/fsnotify/fsnotify
