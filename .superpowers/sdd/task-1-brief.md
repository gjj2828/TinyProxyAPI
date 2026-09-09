# Task 1: 项目初始化

## 概述

创建项目基础结构：go.mod 和 config.toml.example

## Global Constraints（必须遵守）

- Go 1.21+
- 监听地址：`:3178`
- 端点：`POST /v1/responses`
- 配置文件：`config.toml`（TOML 格式）
- 热重载：SIGHUP 信号
- 模型匹配：全量匹配

## 工作目录

`D:/Documents/Projects/TinyProxyAPI`

## 具体任务

### Step 1: 创建 go.mod

```bash
go mod init github.com/user/TinyProxyAPI
```

### Step 2: 创建 config.toml.example

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

### Step 3: 提交

```bash
git init
git add go.mod config.toml.example
git commit -m "feat: initialize project structure"
```

## 验收标准

1. go.mod 文件存在且 module name 正确
2. config.toml.example 文件存在且格式正确
3. git 仓库已初始化
4. 提交已创建
