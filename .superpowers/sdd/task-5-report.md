# Task 5 Report

## Status
DONE

## Commits
- `8aa74aa` - feat: convert to GUI app with system tray support

## Changes Made

### main.go
- 重写为 walk GUI 程序入口
- 使用 `walk.App()` 获取 Application 实例
- 集成托盘和日志窗口
- 通过 `tray.UpdateMenu()`, `tray.ToggleLogWindow()`, `tray.ToggleStartupSetting()`, `tray.CleanupAndExit()` 管理托盘菜单行为
- 使用 `tr.MainWindow().Run()` 启动消息循环

### config/loader.go
- 添加 `SaveConfig(path string, cfg *Config) error` 函数
- 使用 toml.Marshal 序列化配置并写入文件

### tray/tray.go
- 添加 `MainWindow()` 方法返回隐藏的主窗口
- 添加 `UpdateMenu()`, `ToggleLogWindow()`, `ToggleStartupSetting()`, `CleanupAndExit()` 辅助函数
- 添加 config 包导入

### tray/logwindow.go
- 修改 `ShowLogWindow()` 移除不需要的 parent 参数

## Test Summary
- `go build ./...` 编译成功，无错误

## Concerns
- 双击托盘图标会切换日志窗口显示/隐藏（按照需求主窗口是隐藏的）
- 托盘图标路径使用 `filepath.Join(filepath.Dir(os.Args[0]), "icon.ico")` 指向 exe 同目录下
