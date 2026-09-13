# TinyProxyAPI 开源发布准备设计规范 (v0.0.1)

- **日期**: 2026-09-13
- **版本**: v0.0.1
- **作者**: gjj2828
- **目标**: 为 TinyProxyAPI 项目推送到 GitHub 并正式开源提供完整的工程化配置、自动化 CI/CD 及社区规范。

---

## 1. 基础配置与模块规范

### 1.1 开源协议 (License)
- 协议类型: **MIT License**
- 著作权声明: Copyright (c) 2026 gjj2828
- 目标文件: `LICENSE`

### 1.2 Go 模块规范
- 模块路径由 `module github.com/user/TinyProxyAPI` 修改为 `module github.com/gjj2828/TinyProxyAPI`。
- 移除 `go.mod` 末尾临时本地指令 `replace github.com/user/TinyProxyAPI => .`。
- 保证全部内部包引用与模块路径一致。

---

## 2. GitHub Actions 自动化工作流

### 2.1 持续集成 (`.github/workflows/ci.yml`)
- **触发条件**:
  - push 到 `master` / `main`
  - 针对 `master` / `main` 的 pull request
- **运行环境**: `windows-latest`
- **步骤**:
  1. `actions/checkout@v4`
  2. `actions/setup-go@v5` (Go 1.22+ / 稳定版)
  3. `go mod download`
  4. `go test -v ./...`
  5. `go build -ldflags="-H windowsgui" -o TinyProxyAPI.exe .`

### 2.2 自动发布 Release (`.github/workflows/release.yml`)
- **触发条件**: 推送标签 `tags: ['v*']` (例如 `v0.0.1`)
- **运行环境**: `windows-latest`
- **步骤**:
  1. 检出代码并配置 Go 环境。
  2. 编译轻量化二进制产物：
     ```powershell
     go build -ldflags="-H windowsgui -s -w" -o TinyProxyAPI.exe .
     ```
  3. 组织 Release 归档包目录，包含：
     - `TinyProxyAPI.exe`
     - `config.toml.example`
     - `README.md`
     - `README_EN.md`
     - `LICENSE`
  4. 打包为 `TinyProxyAPI-${{ github.ref_name }}-windows-amd64.zip`。
  5. 生成 SHA256 校验和文件。
  6. 使用 `softprops/action-gh-release@v2` 创建 GitHub Release 并上传资产。

---

## 3. 社区规范与文档

### 3.1 GitHub 模板
- **Issue 模板**:
  - `.github/ISSUE_TEMPLATE/bug_report.md` (Bug 报告模板)
  - `.github/ISSUE_TEMPLATE/feature_request.md` (功能提议模板)
- **Pull Request 模板**:
  - `.github/pull_request_template.md`

### 3.2 README 优化 (`README.md` & `README_EN.md`)
- 顶部增加状态徽章：
  - License
  - Go Report / Go Version
  - CI Status
  - GitHub Release
- 增加 Release 下载与免编译快速使用说明。
- 补充 License 章节说明。
