# 托盘左键点击行为实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为系统托盘添加左键单击响应，每次左键点击都触发"显示或激活日志窗口"，单双击行为等价。

**Architecture:** tray 包通过 `MouseDown` 事件监听左键点击，统一触发 `OnLeftClick` handler，不做单双击计时区分。main.go 提供独立的 `activateLogWindow` 函数处理显示/激活逻辑，与菜单项的 `toggleLogWindow` 分离。

**Tech Stack:** Go, walk (Windows GUI), win32 API

## Global Constraints

- 单双击行为等价，不做计时区分
- 菜单项"显示日志"保持切换语义，不受此变更影响
- 删除 `time` 包依赖

---

## Task 1: tray/tray.go — 修改 Handlers 和 Tray 结构体

**Files:**
- Modify: `tray/tray.go`

**Interfaces:**
- Produces: `Handlers.OnLeftClick func()` 新字段；`Tray.OnLeftClick(handler func())` 新方法

- [ ] **Step 1: 修改 tray/tray.go — Handlers 结构体加 OnLeftClick**

在 `Handlers` 结构体中添加 `OnLeftClick` 字段：

```go
type Handlers struct {
    OnToggleLog     func()
    OnLeftClick     func()  // 新增
    OnToggleStartup func()
    OnExit          func()
}
```

- [ ] **Step 2: 修改 tray/tray.go — Tray 结构体删除 onDoubleClick 和 lastClick**

删除 `onDoubleClick func()`、`lastClick time.Time` 字段，删除 `time` 包 import：

```go
type Tray struct {
    trayIcon      *walk.NotifyIcon
    app           *walk.Application
    mainWindow    *walk.MainWindow
    actShowLog    *walk.Action
    actStartup    *walk.Action
    // 删除: onDoubleClick func()
    // 删除: lastClick time.Time
}
```

从 import 中删除 `"time"`。

- [ ] **Step 3: 修改 tray/tray.go — NewTray 中删除 DoubleClicked 注册**

找到 `t.trayIcon.DoubleClicked().Attach(...)` 相关代码，删除整个 `DoubleClicked` 事件注册块（当前是空调用或调用 `t.onDoubleClick`）。

- [ ] **Step 4: 修改 tray/tray.go — 新增 OnLeftClick 方法，替换 OnDoubleClick**

将现有的 `OnDoubleClick` 方法替换为：

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

- [ ] **Step 5: 验证编译**

Run: `go build ./...`
Expected: 无编译错误

- [ ] **Step 6: 提交**

```bash
git add tray/tray.go
git commit -m "refactor: replace OnDoubleClick with OnLeftClick, remove time dependency"
```

---

## Task 2: main.go — 新增 activateLogWindow 并注册 OnLeftClick

**Files:**
- Modify: `main.go`

**Interfaces:**
- Consumes: `tray.Handlers.OnLeftClick`、`logWin.Show()`、`logWin.Activate()`、`tr.SyncMenuState()`
- Produces: `activateLogWindow` 函数

- [ ] **Step 1: 新增 activateLogWindow 函数**

在 `toggleLogWindow` 函数后新增：

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

- [ ] **Step 2: NewTray 注册 OnLeftClick**

找到 `NewTray` 调用，在 `Handlers` 中添加 `OnLeftClick: activateLogWindow`：

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

- [ ] **Step 3: 删除 OnDoubleClick 调用**

找到并删除 `tr.OnDoubleClick(toggleLogWindow)` 这一行（main.go:149 附近）。

- [ ] **Step 4: 验证编译**

Run: `go build ./...`
Expected: 无编译错误

- [ ] **Step 5: 提交**

```bash
git add main.go
git commit -m "feat: add activateLogWindow and register OnLeftClick for tray"
```

---

## Task 3: 手动验证

**Files:**
- None (manual verification)

- [ ] **Step 1: 构建并运行**

Run: `go build -o TinyProxyAPI.exe . && ./TinyProxyAPI.exe`

- [ ] **Step 2: 验证托盘左键单击打开日志窗口**

1. 确认托盘图标显示
2. 日志窗口默认隐藏
3. 左键单击托盘图标 → 日志窗口应显示
4. 窗口显示时再单击托盘 → 窗口应激活并置于前台（不隐藏）

- [ ] **Step 3: 验证托盘菜单不受影响**

1. 右键托盘 → "显示日志" 应仍为切换语义（显示↔隐藏）
2. 确认两个入口独立工作

- [ ] **Step 4: 提交**

```bash
git add -A
git commit -m "test: manual verification of tray left-click behavior"
```

---

## 自审检查

1. **Spec coverage**: spec 中所有变更点都有对应任务 ✓
2. **Placeholder scan**: 无 TBD/TODO ✓
3. **Type consistency**: `Handlers.OnLeftClick`、`Tray.OnLeftClick`、`activateLogWindow` 名称全流程一致 ✓
