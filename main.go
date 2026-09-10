package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/user/TinyProxyAPI/config"
	"github.com/user/TinyProxyAPI/handler"
	"github.com/user/TinyProxyAPI/matcher"
	"github.com/user/TinyProxyAPI/tray"

	"github.com/lxn/walk"
)

var (
	conf       *config.Config
	tr         *tray.Tray
	logWin     *tray.LogWindow
	configPath string
)

func main() {
	configPathFlag := flag.String("config", "config.toml", "path to config file")
	flag.Parse()
	configPath = *configPathFlag

	var err error
	conf, err = config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// 创建 walk 应用
	app := walk.App()

	// 创建日志窗口
	logWin, err = tray.ShowLogWindow()
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

	// 初始化托盘菜单
	tray.UpdateMenu(tr, logWin, conf, configPath)

	tr.OnDoubleClick(func() {
		// 主窗口隐藏，双击托盘只切换日志窗口
		tray.ToggleLogWindow(logWin, tr, conf, configPath)
	})

	// 启动 HTTP 服务器
	m := matcher.NewMatcher(conf)
	h := handler.NewProxyHandler(m)

	go func() {
		log.Printf("server listening on %s", conf.Server.Addr)
		if err := http.ListenAndServe(conf.Server.Addr, h); err != nil {
			log.Printf("server error: %v", err)
		}
	}()

	// 运行消息循环
	tr.MainWindow().Run()
}
