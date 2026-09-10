# 托盘左键点击行为设计

## 概述

为系统托盘添加左键单击响应：打开或激活日志窗口，与 Microsoft 托盘交互建议对齐。

## 背景

当前托盘图标左键单击无任何响应，不符合 Windows 用户习惯（单击托盘图标通常打开相关窗口）。

## 行为定义

| 入口 | 行为 |
|------|------|
| **托盘左键单击** | 日志窗口隐藏 → 显示；日志窗口已显示 → 激活并置于前台 |
| **托盘左键双击** | 同左键单击（两者等价，不区分） |
| **托盘菜单 → "显示日志"** | 保持原有的切换语义：显示 ↔ 隐藏 |

**设计原则**：单击/双击等价，不需要区分。菜单项保持独立的切换语义。

## 变更点

### 1. tray/tray.go

**Handlers 结构体**新增 `OnLeftClick` 字段：

```go
type Handlers struct {
    OnToggleLog     func()  // 菜单"显示日志"：切换显示/隐藏
    OnLeftClick     func()  // 新增：左键单击/双击：仅显示/激活
    OnToggleStartup func()
    OnExit          func()
}
```

**Tray 结构体**删除 `onDoubleClick`、`lastClick` 字段，删除 `time` 包依赖：

```go
type Tray struct {
    trayIcon      *walk.NotifyIcon
    app           *walk.Application
    mainWindow    *walk.MainWindow
    actShowLog    *walk.Action
    actStartup    *walk.Action
    // 删除 onDoubleClick func()
    // 删除 lastClick time.Time
}
```

**OnLeftClick 方法**替换 `OnDoubleClick`，每次左键点击都直接触发，不计时区分单双击：

```go
// OnLeftClick 注册左键单击/双击处理函数（两者等价）。
// 每次左键按下都触发 handler，不做单双击计时区分。
func (t *Tray) OnLeftClick(handler func()) {
    t.trayIcon.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
        if button == walk.LeftButton && handler != nil {
            handler()
        }
    })
}
```

### 2. main.go

**新增 `activateLogWindow`**：

```go
// activateLogWindow 将日志窗口显示并激活到前台。
// 日志窗口隐藏时调用 Show()；已显示时调用 Activate() 置于前台。
// 菜单勾选框由调用方同步。
func activateLogWindow() {
    if !logWin.IsVisible() {
        logWin.Show()
        tr.SyncMenuState(true, st.ShowLogOnStartup)
    } else {
        logWin.Activate()
    }
}
```

**NewTray 注册**：

```go
tr, err = tray.NewTray(app, tray.Handlers{
    OnToggleLog:    toggleLogWindow,
    OnLeftClick:    activateLogWindow,
    OnToggleStartup: func() {
        st.ShowLogOnStartup = !st.ShowLogOnStartup
        if saveErr := state.Save(statePath, st); saveErr != nil {
            log.Printf("failed to save state: %v", saveErr)
        }
        tr.SyncMenuState(logWin.IsVisible(), st.ShowLogOnStartup)
    },
    OnExit: func() {
        if tr != nil {
            tr.Destroy()
        }
        if logWin != nil {
            logWin.Hide()
        }
        os.Exit(0)
    },
})
```

**删除 `tr.OnDoubleClick(toggleLogWindow)`**，替换为（若 NewTray 内部已注册，无需外部调用）：

```go
tr.OnLeftClick(activateLogWindow)
```

注：若 `OnLeftClick` 在 `NewTray` 内部注册，则不需要在 main.go 单独调用。

## 文件变更

```
tray/tray.go    # Handlers 加 OnLeftClick，OnDoubleClick 改 OnLeftClick，删除 lastClick 和 time 依赖
main.go         # 新增 activateLogWindow，NewTray 注册 OnLeftClick
```

## 依赖变更

无新增依赖。删除 `time` 包在 tray/tray.go 中的使用。
