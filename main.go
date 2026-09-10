package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/user/TinyProxyAPI/config"
	"github.com/user/TinyProxyAPI/handler"
	"github.com/user/TinyProxyAPI/matcher"
	"github.com/user/TinyProxyAPI/signal"
	"github.com/user/TinyProxyAPI/tray"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

var (
	conf       *config.Config
	tr         *tray.Tray
	logWin     *tray.LogWindow
	configPath string
)

func showError(title string, msg string) {
	walk.MsgBox(nil, title, msg, walk.MsgBoxIconError)
}

func main() {
	configPathFlag := flag.String("config", "config.toml", "path to config file")
	flag.Parse()
	configPath = *configPathFlag

	// 确保配置路径是绝对路径，优先使用 exe 所在目录
	if !filepath.IsAbs(configPath) {
		if filepath.Dir(os.Args[0]) != "" {
			configPath = filepath.Join(filepath.Dir(os.Args[0]), configPath)
		}
	}

	var err error
	conf, err = config.LoadConfig(configPath)
	if err != nil {
		showError("配置加载失败", fmt.Sprintf("failed to load config: %v\n\n路径: %s", err, configPath))
		return
	}

	// 创建 walk 应用
	app := walk.App()

	// 创建日志窗口
	logWin, err = tray.ShowLogWindow()
	if err != nil {
		showError("日志窗口创建失败", fmt.Sprintf("failed to create log window: %v", err))
		return
	}

	// 根据配置决定是否在启动时显示日志窗口
	if conf.UI.ShowLogOnStartup {
		logWin.Show()
	}

	// 将日志输出重定向到日志窗口
	log.SetOutput(&logWriter{logWin: logWin})
	log.SetFlags(log.LstdFlags)

	// 创建托盘
	iconPath := filepath.Join(filepath.Dir(os.Args[0]), "icon.ico")
	tr, err = tray.NewTray(app, iconPath)
	if err != nil {
		showError("托盘创建失败", fmt.Sprintf("failed to create tray: %v", err))
		return
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

	// 热重载：合并两种重载触发 SIGHUP（Linux/Mac）+ 配置文件变化（跨平台）。
	// 任一来源触发都会通过同一个 channel 通知重载逻辑。
	reload := make(chan struct{}, 1)

	hup := signal.Watch()
	go func() {
		for range hup {
			select {
			case reload <- struct{}{}:
			default:
			}
		}
	}()

	if fileCh, err := signal.WatchConfig(configPath); err != nil {
		log.Printf("warn: file watcher disabled: %v", err)
	} else {
		go func() {
			for range fileCh {
				select {
				case reload <- struct{}{}:
				default:
				}
			}
		}()
	}

	// 处理热重载信号
	go func() {
		for range reload {
			log.Println("reloading config...")
			newCfg, err := config.LoadConfig(configPath)
			if err != nil {
				log.Printf("failed to reload config: %v", err)
				continue
			}
			// 复用现有 handler，仅原子替换其 matcher 指针。
			// 这样 HTTP server 持有的 handler 永远不变，但 ServeHTTP
			// 内部每次都会 Load 最新 matcher，新请求立即走新配置。
			h.Update(matcher.NewMatcher(newCfg))
			conf = newCfg
			tray.UpdateMenu(tr, logWin, conf, configPath)
			log.Println("config reloaded successfully")
		}
	}()

	// 运行消息循环
	tr.MainWindow().Run()
}

// Dummy use of win package to avoid unused import error
var _ = win.MB_OK

// logWriter 实现 io.Writer，将日志写入日志窗口
type logWriter struct {
	logWin *tray.LogWindow
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	if w.logWin != nil {
		w.logWin.AppendText(string(p))
	}
	return len(p), nil
}
