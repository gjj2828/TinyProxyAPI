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
| 右键 → 显示日志 | 复选项，点击切换日志窗口显示状态，勾选状态实时同步 |
| 右键 → 日志启动时显示 | 复选项，点击切换勾选状态并实时保存到配置文件 |
| 右键 → 退出 | 关闭日志窗口，销毁托盘，退出程序 |

**启动行为**：根据配置决定启动时是否显示日志窗口。

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

统一使用复选项（Checkable），菜单状态与实际状态实时同步：

| 菜单项 | 类型 | 功能 |
|--------|------|------|
| 显示日志 | 复选项 | 勾选 = 日志窗口当前显示；点击切换显示/隐藏 |
| 日志启动时显示 | 复选项 | 勾选 = 下次启动时自动显示日志窗口；点击切换并实时保存到配置文件 |
| 退出 | 普通项 | 关闭程序 |

### 3. 日志窗口

### 3. 日志窗口

- 标题：`TinyProxyAPI - 日志`
- 大小：600x400，居中屏幕
- 只读多行文本框，自动滚动
- 关闭按钮隐藏窗口但不退出程序

### 4. 配置文件新增项

```toml
[ui]
# 启动时是否显示日志窗口（默认 false）
show_log_on_startup = false
```

**注意**：托盘中"日志启动时显示"菜单项的勾选状态需要实时同步写入配置文件，以便下次启动生效。

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
2. 配置文件添加 `[ui] show_log_on_startup` 项
3. 修改 `main.go` 为 GUI 程序入口
4. 创建 `tray/tray.go` 实现托盘逻辑
5. 创建 `tray/logwindow.go` 实现日志窗口
6. 修改构建命令添加 `-ldflags="-H windowsgui"`
