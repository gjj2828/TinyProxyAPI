# TinyProxyAPI 开源发布准备实施计划

- **设计文档**: [`docs/superpowers/specs/2026-09-13-github-opensource-preparation-design.md`](file:///D:/Documents/Projects/TinyProxyAPI/docs/superpowers/specs/2026-09-13-github-opensource-preparation-design.md)
- **目标版本**: `v0.0.1`

---

## 任务拆解与执行步骤

### Task 1: 补充开源许可证与修正 Go 模块路径
- [ ] 创建 `LICENSE` (MIT License, Copyright (c) 2026 gjj2828)。
- [ ] 修改 `go.mod`：将 module 名称改为 `github.com/gjj2828/TinyProxyAPI`，移除末尾 `replace` 指令。
- [ ] 检查代码中内部 import 并执行 `go mod tidy` / `go test ./...` 确保所有测试通过。

### Task 2: 配置 GitHub Actions 自动化工作流
- [ ] 创建 `.github/workflows/ci.yml`（Windows 环境下的测试与构建验证）。
- [ ] 创建 `.github/workflows/release.yml`（推送 `v*` tag 时自动编译 Windows zip 包并发布 Release）。

### Task 3: 添加 GitHub 社区 Issue / PR 模板
- [ ] 创建 `.github/ISSUE_TEMPLATE/bug_report.md`。
- [ ] 创建 `.github/ISSUE_TEMPLATE/feature_request.md`。
- [ ] 创建 `.github/pull_request_template.md`。

### Task 4: 优化 README 文档与徽章
- [ ] 在 `README.md` 和 `README_EN.md` 顶部添加 License、Go Version、CI、Release 徽章。
- [ ] 增加 GitHub Release 下载说明和 License 声明。

### Task 5: 整体测试与最终验证
- [ ] 运行 `go test ./...` 确保所有单元测试通过。
- [ ] 运行 `.\build.ps1` 验证本地构建正常。
- [ ] 提交所有改动到 Git 仓库。
