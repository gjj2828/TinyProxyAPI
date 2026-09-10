# Task 6 Report: 更新构建配置

## Status

**DONE**

## Commits

- `c1235b1` - docs: add build.ps1 with -ldflags=-H windowsgui to hide console window

## Implementation Summary

项目原本没有构建脚本，需要手动执行 `go build` 命令。为确保构建时隐藏控制台窗口（使用 `-ldflags="-H windowsgui"`），创建了 `build.ps1` 构建脚本。

### 创建的文件

- `build.ps1` - PowerShell 构建脚本，包含正确的构建命令：
  ```powershell
  go build -ldflags="-H windowsgui" -o TinyProxyAPI.exe
  ```

## Test Summary

- 构建脚本已创建并添加到 Git
- 无需运行测试（此任务仅涉及构建配置）

## Concerns

无
