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

