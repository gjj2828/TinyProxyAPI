# TinyProxyAPI 系统托盘功能设计

## 概述

为 TinyProxyAPI 添加系统托盘功能，使程序运行时隐藏控制台窗口，用户通过系统托盘图标管理程序。

## 架构变更

将程序从控制台程序改为 GUI 程序（隐藏主窗口 + 系统托盘）：

```
原架构：                    新架构：
┌─────────┐               ┌──────────────────┐
│ Console │               │  Hidden Window   │◄──── 托盘图标宿主
└─────────┘               └────────┬─────────┘
                                   │
                            ┌──────┴──────┐
                            │  System Tray │
                            │  - 图标      │
                            │  - 右键菜单  │
                            │  - 双击恢复  │
                            └─────────────┘
```

## 托盘交互行为

| 操作 | 行为 |
|------|------|
| 点击最小化按钮 | 窗口隐藏，仅在托盘显示图标 |
| 双击托盘图标 | 窗口显示并激活 |
| 右键 → 显示日志 | 打开日志查看窗口 |
| 右键 → 退出 | 关闭日志窗口，销毁托盘，退出程序 |

## 组件设计

### 1. tray 包

**tray/tray.go**
- `NewTray(app *walk.App) (*Tray, error)` — 创建托盘实例
- `SetIcon(iconPath string)` — 设置托盘图标
- `SetTooltip(text string)` — 设置鼠标悬停提示
- `OnDoubleClick(handler func())` — 双击事件回调
- `Destroy()` — 销毁托盘

**tray/logwindow.go**
- `ShowLogWindow(parent walk.MainWindow) error` — 显示日志窗口
- 窗口关闭时只隐藏，不销毁

### 2. 托盘右键菜单

| 菜单项 | 功能 |
|--------|------|
| 显示日志 | 打开日志查看窗口 |
| 退出 | 关闭程序 |

### 3. 日志窗口

- 标题：`TinyProxyAPI - 日志`
- 大小：600x400，居中屏幕
- 只读多行文本框，自动滚动
- 关闭按钮隐藏窗口但不退出程序

## 依赖

- `github.com/lxn/walk` — Windows GUI 库
- `github.com/lxn/win` — Windows API 封装（walk 依赖）

## 构建配置

编译时使用 `-ldflags="-H windowsgui"` 隐藏控制台窗口。

## 文件变更

```
TinyProxyAPI/
├── main.go              # 改为 GUI 程序入口
├── tray/
│   ├── tray.go         # 新增：托盘管理
│   └── logwindow.go    # 新增：日志窗口
├── config/
├── handler/
├── matcher/
├── signal/
├── go.mod
└── config.toml
```

## 实现步骤

1. 添加 `walk` 依赖
2. 修改 `main.go` 为 GUI 程序入口
3. 创建 `tray/tray.go` 实现托盘逻辑
4. 创建 `tray/logwindow.go` 实现日志窗口
5. 修改构建命令添加 `-ldflags="-H windowsgui"`
