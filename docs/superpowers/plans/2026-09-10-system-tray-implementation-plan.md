# TinyProxyAPI 系统托盘实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 TinyProxyAPI 添加系统托盘功能，程序启动时隐藏控制台窗口，用户通过托盘图标管理程序（显示/隐藏日志窗口、退出）。

**Architecture:** 将程序从控制台程序改为 GUI 程序（使用 walk 库创建隐藏主窗口作为托盘图标宿主），托盘菜单使用两个复选项（显示日志、日志启动时显示）和一个普通退出按钮。

**Tech Stack:** Go 1.26.5, github.com/lxn/walk, github.com/lxn/win

## Global Constraints

- Go 版本：1.26.5
- 隐藏控制台窗口：编译时使用 `-ldflags="-H windowsgui"`
- 配置文件格式：TOML，新增 `[ui]` section
- 托盘菜单项：两个 Checkable 菜单项 + 一个普通退出项

---

## 文件结构

```
TinyProxyAPI/
├── main.go              # 改为 GUI 程序入口，使用 walk.MsgLoop
├── tray/
│   ├── tray.go         # 新增：托盘管理（图标、菜单、双击事件）
│   └── logwindow.go    # 新增：日志查看窗口
├── config/
│   └── loader.go       # 修改：添加 UI 配置 struct 和解析
├── go.mod              # 修改：添加 walk 依赖
└── config.toml         # 修改：添加 [ui] section
```

---

## Task 1: 添加 walk 依赖

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: 添加 walk 依赖**

Run: `go get github.com/lxn/walk@latest`
Expected: go.mod 添加 `github.com/lxn/walk v0.8.0` 和 `github.com/lxn/win v0.21.0`

- [ ] **Step 2: 提交**

```bash
git add go.mod go.sum
git commit -m "deps: add walk for system tray support"
```

---

## Task 2: 添加 UI 配置

**Files:**
- Modify: `config/loader.go`
- Modify: `config.toml`

**Interfaces:**
- Consumes: toml 配置文件
- Produces: `Config.UI.ShowLogOnStartup` 字段（bool 类型，默认为 false）

- [ ] **Step 1: 在 Config 结构体中添加 UI 配置**

在 `config/loader.go` 中添加：

```go
type Config struct {
    Server      ServerConfig
    Default     DefaultConfig
    AllowedKeys []string  `toml:"allowed_keys"`
    Platforms  map[string]Platform
    Models     map[string]string
    UI         UIConfig
}

type UIConfig struct {
    ShowLogOnStartup bool `toml:"show_log_on_startup"`
}
```

- [ ] **Step 2: 在 config.toml 中添加 [ui] section**

```toml
[ui]
# 启动时是否显示日志窗口（默认 false）
show_log_on_startup = false
```

- [ ] **Step 3: 提交**

```bash
git add config/loader.go config.toml
git commit -m "config: add [ui] section with show_log_on_startup"
```

---

## Task 3: 创建托盘管理（tray/tray.go）

**Files:**
- Create: `tray/tray.go`

**Interfaces:**
- Consumes: `*walk.App`，托盘图标路径
- Produces:
  - `Tray` struct
  - `NewTray(app *walk.App, iconPath string) (*Tray, error)`
  - `SetTooltip(text string)`
  - `SetMenu(items []MenuItem)` — 设置菜单项
  - `OnDoubleClick(handler func())`
  - `Destroy()`

- [ ] **Step 1: 实现 Tray 结构体和 NewTray 函数**

```go
// tray/tray.go
package tray

import (
	"github.com/lxn/win"
	"github.com/lxn/walk"
)

type Tray struct {
	trayIcon       *walk.NotifyIcon
	app            *walk.App
	onDoubleClick  func()
}

type MenuItem struct {
	Text      string
	Checked   bool
	OnClick   func()
}

func NewTray(app *walk.App, iconPath string) (*Tray, error) {
	t := &Tray{app: app}

	ni, err := walk.NewNotifyIcon(t.app)
	if err != nil {
		return nil, err
	}
	t.trayIcon = ni

	if err := t.trayIcon.SetIcon(iconPath); err != nil {
		return nil, err
	}

	return t, nil
}

func (t *Tray) SetTooltip(text string) {
	t.trayIcon.SetToolTip(text)
}

func (t *Tray) SetMenu(items []MenuItem) {
	menu, _ := walk.NewMenu()
	for _, item := range items {
		action := walk.NewAction()
		action.SetText(item.Text)
		action.SetChecked(item.Checked)
		action.Clicked().Attach(item.OnClick)
		menu.Actions().Add(action)
	}
	t.trayIcon.SetContextMenu(menu)
}

func (t *Tray) OnDoubleClick(handler func()) {
	t.onDoubleClick = handler
	t.trayIcon.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton() && t.onDoubleClick != nil {
			t.onDoubleClick()
		}
	})
}

func (t *Tray) Destroy() {
	t.trayIcon.Dispose()
}
```

- [ ] **Step 2: 提交**

```bash
git add tray/tray.go
git commit -m "feat: add tray management module"
```

---

## Task 4: 创建日志窗口（tray/logwindow.go）

**Files:**
- Create: `tray/logwindow.go`

**Interfaces:**
- Consumes: 父窗口 `walk.MainWindow`
- Produces:
  - `LogWindow` struct
  - `ShowLogWindow(parent walk.MainWindow) (*LogWindow, error)`
  - `(*LogWindow).AppendText(text string)` — 追加日志文本
  - `(*LogWindow).IsVisible() bool`
  - `(*LogWindow).Show() / Hide()`

- [ ] **Step 1: 实现 LogWindow 结构体**

```go
// tray/logwindow.go
package tray

import (
	"github.com/lxn/walk"
	"github.com/lxn/win"
)

type LogWindow struct {
	*walk.MainWindow
	textEdit *walk.TextEdit
}

func ShowLogWindow(parent walk.MainWindow) (*LogWindow, error) {
	ml, err := walk.NewMainWindow(parent)
	if err != nil {
		return nil, err
	}

	ml.SetTitle("TinyProxyAPI - 日志")
	ml.SetWidth(600)
	ml.SetHeight(400)
	ml.SetLayout(walk.NewVBoxLayout())
	ml.Center()

	te, err := walk.NewTextEdit(ml)
	if err != nil {
		return nil, err
	}
	te.SetReadOnly(true)
	te.SetVerticalScrollBarPolicy(walk.ScrollBarAlwaysOn)

	ml.SetSize(600, 400)

	lw := &LogWindow{
		MainWindow: ml,
		textEdit:   te,
	}

	ml.Closing().Attach(func(cancel *bool) {
		*cancel = true
		lw.Hide()
	})

	return lw, nil
}

func (lw *LogWindow) AppendText(text string) {
	walk.NewGoroutineProperty().Enqueue(func() {
		lw.textEdit.AppendText(text)
	})
}

func (lw *LogWindow) IsVisible() bool {
	return lw.Visible()
}

func (lw *LogWindow) Show() {
	walk.NewGoroutineProperty().Enqueue(func() {
		lw.Show()
		lw.Activate()
	})
}

func (lw *LogWindow) Hide() {
	walk.NewGoroutineProperty().Enqueue(func() {
		lw.Hide()
	})
}
```

- [ ] **Step 2: 提交**

```bash
git add tray/logwindow.go
git commit -m "feat: add log window module"
```

---

## Task 5: 修改 main.go 为 GUI 程序入口

**Files:**
- Modify: `main.go`

**Interfaces:**
- Consumes: `config.Config`，日志窗口，托盘实例
- Produces: 完整的 GUI 程序入口

- [ ] **Step 1: 重写 main.go 为 walk GUI 程序**

```go
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/user/TinyProxyAPI/config"
	"github.com/user/TinyProxyAPI/handler"
	"github.com/user/TinyProxyAPI/matcher"
	"github.com/user/TinyProxyAPI/signal"
	"github.com/user/TinyProxyAPI/tray"

	"github.com/lxn/walk"
)

var (
	conf    *config.Config
	app     *walk.App
	tr       *tray.Tray
	logWin  *tray.LogWindow
	logWinMux sync.RWMutex
	cfgMux   sync.RWMutex
)

func main() {
	configPath := flag.String("config", "config.toml", "path to config file")
	flag.Parse()

	var err error
	conf, err = config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 创建 walk 应用
	app, err := walk.NewApp()
	if err != nil {
		log.Fatalf("failed to create app: %v", err)
	}

	// 创建日志窗口
	logWin, err = tray.ShowLogWindow(nil)
	if err != nil {
		log.Fatalf("failed to create log window: %v", err)
	}

	// 根据配置决定是否在启动时显示日志窗口
	if conf.UI.ShowLogOnStartup {
		logWin.Show()
	}

	// 创建托盘
	iconPath := filepath.Join(filepath.Dir(os.Args[0]), "icon.ico")
	tr, err = tray.NewTray(app, iconPath)
	if err != nil {
		log.Fatalf("failed to create tray: %v", err)
	}
	tr.SetTooltip("TinyProxyAPI")

	// 设置托盘菜单
	updateTrayMenu := func() {
		logWinMux.RLock()
		showLogChecked := logWin != nil && logWin.IsVisible()
		logWinMux.RUnlock()

		cfgMux.RLock()
		startupChecked := conf.UI.ShowLogOnStartup
		cfgMux.RUnlock()

		items := []tray.MenuItem{
			{
				Text:    "显示日志",
				Checked: showLogChecked,
				OnClick: func() {
					toggleLogWindow()
				},
			},
			{
				Text:    "日志启动时显示",
				Checked: startupChecked,
				OnClick: func() {
					toggleStartupSetting()
				},
			},
			{
				Text: "退出",
				OnClick: func() {
					cleanupAndExit()
				},
			},
		}
		tr.SetMenu(items)
	}

	tr.OnDoubleClick(func() {
		showMainWindow()
	})

	updateTrayMenu()

	// 启动 HTTP 服务器
	m := matcher.NewMatcher(conf)
	h := handler.NewProxyHandler(m)

	go func() {
		log.Printf("server listening on %s", conf.Server.Addr)
		if err := http.ListenAndServe(conf.Server.Addr, h); err != nil {
			log.Printf("server error: %v", err)
		}
	}()

	// 启动后更新菜单状态
	walk.NewGoroutineProperty().Enqueue(func() {
		updateTrayMenu()
	})

	// 运行消息循环
	app.Run()
}

func toggleLogWindow() {
	logWinMux.Lock()
	defer logWinMux.Unlock()

	if logWin.IsVisible() {
		logWin.Hide()
	} else {
		logWin.Show()
	}
	updateTrayMenu()
}

func toggleStartupSetting() {
	cfgMux.Lock()
	conf.UI.ShowLogOnStartup = !conf.UI.ShowLogOnStartup
	cfgMux.Unlock()

	// 保存配置
	if err := config.SaveConfig(*configPath, conf); err != nil {
		log.Printf("failed to save config: %v", err)
	}
	updateTrayMenu()
}

func showMainWindow() {
	// 主窗口隐藏，不需要显示
}

func cleanupAndExit() {
	if tr != nil {
		tr.Destroy()
	}
	if logWin != nil {
		logWin.Hide()
	}
	os.Exit(0)
}
```

- [ ] **Step 2: 在 config/loader.go 中添加 SaveConfig 函数**

```go
func SaveConfig(path string, cfg *Config) error {
	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
```

- [ ] **Step 3: 提交**

```bash
git add main.go config/loader.go
git commit -m "feat: convert to GUI app with system tray support"
```

---

## Task 6: 更新构建配置

**Files:**
- Modify: 构建脚本（如有）

- [ ] **Step 1: 确保构建命令使用 `-ldflags="-H windowsgui"`**

如果使用 go build，确保命令为：
```bash
go build -ldflags="-H windowsgui" -o TinyProxyAPI.exe
```

---

## 实现顺序

1. Task 1: 添加 walk 依赖
2. Task 2: 添加 UI 配置
3. Task 3: 创建托盘管理
4. Task 4: 创建日志窗口
5. Task 5: 修改 main.go
6. Task 6: 更新构建配置

---

## 注意事项

- walk 库需要在 Windows 环境编译
- 托盘图标需要准备 `icon.ico` 文件放在 exe 同目录，或内嵌资源
- 菜单项的勾选状态需要通过 `updateTrayMenu()` 实时刷新
- 配置文件保存使用 toml.Marshal，需要引入 toml 包
