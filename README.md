# TinyProxyAPI

[English](./README_EN.md) | 极轻量的 API 路由代理服务，根据请求中的模型名将请求透传到对应的平台。

## 特性

- **透传** - 使用 `httputil.ReverseProxy`，最小化 Header 和 Body 修改
- **灵活路由** - 模型名全量匹配，通过 TOML 配置
- **平台抽象** - 同一平台的多个模型共享 base_url 和 api_key
- **热重载** - 改完 `config.toml` 保存即生效（基于 fsnotify 跨平台监听）；Linux/Mac 也可发 SIGHUP 触发
- **系统托盘** - Windows 运行时隐藏控制台窗口，通过托盘图标管理程序（显示/隐藏日志窗口、退出）
- **日志窗口** - 可查看程序运行日志，支持启动时自动显示

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

```powershell
# Windows 构建（隐藏控制台窗口）
.\build.ps1

# 运行
.\TinyProxyAPI.exe -config config.toml
```

> 或手动构建：`go build -ldflags="-H windowsgui" -o TinyProxyAPI.exe .`
> Windows 上 `go build -o name .` 不会自动补 `.exe` 后缀，必须显式指定。

### 请求示例

**模型列表：**
```bash
curl http://localhost:3178/v1/models
```

**聊天补全：**
```bash
curl -X POST http://localhost:3178/v1/chat/completions \
  -H "Content-Type: application/json" \
  -d '{"model": "gpt-4o", "messages": [{"role": "user", "content": "Hello"}]}'
```

**其他 OpenAI 兼容接口**（透传到对应平台）：
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

## 系统托盘（Windows）

程序启动后会在系统托盘显示图标，右键菜单提供以下功能：

| 菜单项 | 功能 |
|--------|------|
| 显示日志 | 勾选后显示日志查看窗口 |
| 日志启动时显示 | 勾选后下次启动自动显示日志窗口 |
| 退出 | 关闭程序 |

- 单击托盘图标可切换日志窗口显示状态（替代了旧版的双击）
- 点击最小化按钮会隐藏窗口到托盘

### 界面截图

右键单击系统托盘图标，可以显示日志、设置是否在启动时自动显示日志，或退出程序：

![TinyProxyAPI 托盘菜单](./docs/screenshots/tray-menu.png)

日志窗口会显示服务器启动、配置热重载以及运行时错误等信息：

![TinyProxyAPI 日志窗口](./docs/screenshots/log-window.png)

### 界面状态

「日志启动时显示」的设置**不由 `config.toml` 管理**，而是由托盘菜单控制，保存在用户目录的 `~/.tinyproxyapi/state.json` 中，默认值为启动时显示。

`config.toml` 始终只读——程序不会写回该文件，你的注释和格式不会被改动。

### 重新生成图标

托盘图标由 `tools/genicon` 生成并嵌入 `rsrc.syso`，单文件分发即可，无需准备外部图标文件：

```bash
go run ./tools/genicon
rsrc -manifest TinyProxyAPI.exe.manifest -ico icon.ico -o rsrc.syso
```

生成产物 `icon.ico` 与 `rsrc.syso` 需要一并提交。

> **注意**：`rsrc` 分配的 GROUP_ICON 资源 ID 取决于 ICO 内的图像条目数，不是固定值。改动 `tools/genicon/` 后如果条目数变化，必须更新 `tray/tray.go` 里的 `iconResourceID` 常量——构建 exe 后用 `go run ./tools/resid TinyProxyAPI.exe` 读出实际 ID 再核对。

## 项目结构

```
TinyProxyAPI/
├── main.go            # 程序入口（托盘编排、状态管理）
├── config/
│   └── loader.go      # 配置加载（只读，不写回 config.toml）
├── state/
│   └── state.go       # 界面状态读写（state.json）
├── handler/
│   └── proxy.go       # 请求处理
├── matcher/
│   └── matcher.go     # 路由匹配
├── signal/
│   ├── sighup.go      # SIGHUP 信号处理（Linux/Mac）
│   └── watcher.go     # fsnotify 文件监听（跨平台）
├── tray/
│   ├── tray.go        # 系统托盘管理
│   └── logwindow.go   # 日志窗口
├── tools/
│   ├── genicon/       # 生成 icon.ico
│   └── resid/         # 从 exe 读出图标资源 ID
├── config.toml        # 配置文件（含密钥，不入库）
├── config.toml.example  # 配置模板
├── icon.ico           # 图标资源，已嵌入 rsrc.syso
├── rsrc.syso          # 内嵌资源（图标 + manifest），构建时自动链接
├── build.ps1          # Windows 构建脚本
└── TinyProxyAPI.exe.manifest # 公共控件 manifest
```

## 依赖

- Go 1.21+
- github.com/pelletier/go-toml/v2
- github.com/fsnotify/fsnotify
- github.com/lxn/walk（系统托盘）
