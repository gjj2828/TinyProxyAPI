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
	"github.com/user/TinyProxyAPI/state"
	"github.com/user/TinyProxyAPI/tray"

	"github.com/lxn/walk"
)

var (
	conf       *config.Config
	tr         *tray.Tray
	logWin     *tray.LogWindow
	configPath string
	st         *state.State
	statePath  string
)

func showError(title string, msg string) {
	walk.MsgBox(nil, title, msg, walk.MsgBoxIconError)
}

// toggleLogWindow 切换日志窗口可见性，并同步托盘菜单勾选状态。
// 菜单"显示日志"项与双击托盘共用这一份逻辑。
func toggleLogWindow() {
	if logWin.IsVisible() {
		logWin.Hide()
	} else {
		logWin.Show()
	}
	tr.SyncMenuState(st.ShowLogOnStartup, logWin.IsVisible())
}

func main() {
	configPathFlag := flag.String("config", "config.toml", "path to config file")
	flag.Parse()
	configPath = *configPathFlag

	// exe 目录：从 PATH 启动时 os.Args[0] 可能只是文件名，用 os.Executable 取绝对路径
	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)

	// 确保配置路径是绝对路径，优先使用 exe 所在目录
	if !filepath.IsAbs(configPath) {
		if exeDir != "" {
			configPath = filepath.Join(exeDir, configPath)
		}
	}

	var err error
	conf, err = config.LoadConfig(configPath)
	if err != nil {
		showError("配置加载失败", fmt.Sprintf("failed to load config: %v\n\n路径: %s", err, configPath))
		return
	}

	// 界面状态独立存于 exe 同目录，不碰 config.toml
	if exeDir != "" {
		statePath = filepath.Join(exeDir, "state.json")
	}
	st = state.Load(statePath)

	// 创建 walk 应用
	app := walk.App()

	// 创建日志窗口
	logWin, err = tray.ShowLogWindow()
	if err != nil {
		showError("日志窗口创建失败", fmt.Sprintf("failed to create log window: %v", err))
		return
	}

	// 根据界面状态决定是否在启动时显示日志窗口
	if st.ShowLogOnStartup {
		logWin.Show()
	}

	// 将日志输出重定向到日志窗口
	log.SetOutput(&logWriter{logWin: logWin})
	log.SetFlags(log.LstdFlags)

	// 创建托盘；菜单在 NewTray 内部一次性构建，编排逻辑全部在本文件
	tr, err = tray.NewTray(app, tray.Handlers{
		OnToggleLog: toggleLogWindow,
		OnToggleStartup: func() {
			st.ShowLogOnStartup = !st.ShowLogOnStartup
			if saveErr := state.Save(statePath, st); saveErr != nil {
				log.Printf("failed to save state: %v", saveErr)
			}
			tr.SyncMenuState(st.ShowLogOnStartup, logWin.IsVisible())
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
	if err != nil {
		showError("托盘创建失败", fmt.Sprintf("failed to create tray: %v", err))
		return
	}
	tr.SetTooltip("TinyProxyAPI")

	// 建立初始勾选状态（NewTray 拿不到状态值，故在此设定一次）
	tr.SyncMenuState(st.ShowLogOnStartup, logWin.IsVisible())

	// 主窗口隐藏，双击托盘只切换日志窗口
	tr.OnDoubleClick(toggleLogWindow)

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
			log.Println("config reloaded successfully")
		}
	}()

	// 运行消息循环
	tr.MainWindow().Run()
}

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
