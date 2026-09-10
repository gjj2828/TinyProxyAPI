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
	tr.SyncMenuState(logWin.IsVisible(), st.ShowLogOnStartup)
}

func main() {
	configPathFlag := flag.String("config", "config.toml", "path to config file")
	flag.Parse()
	configPath = *configPathFlag

	// exe 目录：从 PATH 启动时 os.Args[0] 可能只是文件名，用 os.Executable 取绝对路径。
	// 失败时退回 configPath 所在目录——两者共用同一段逻辑，保证始终落在
	// 一个绝对路径上，而不是隐式解析到 os.Getwd()。
	//
	// 回退分支必须走 filepath.Abs：filepath.Dir(configPath) 对 "config.toml"
	// 返回 "."，得到的是相对路径，与本次修复要消除的 cwd 相对缺陷同类。Abs 显式
	// 锚定到 cwd，让回退至少是确定的绝对路径。
	var baseDir string
	if exe, exeErr := os.Executable(); exeErr == nil {
		baseDir = filepath.Dir(exe)
	} else {
		fallbackDir, absErr := filepath.Abs(filepath.Dir(configPath))
		if absErr != nil {
			// Abs 仅在路径为空或 cwd 取不到时失败；此时退回系统临时目录，
			// 保证仍是绝对路径，绝不落入 cwd 相对缺陷。
			baseDir = os.TempDir()
			log.Printf("warn: os.Executable and filepath.Abs both failed, using temp dir %s: exe=%v abs=%v", baseDir, exeErr, absErr)
		} else {
			log.Printf("warn: os.Executable failed, using cwd-relative location %s instead of exe dir: %v", fallbackDir, exeErr)
			baseDir = fallbackDir
		}
	}

	// 确保配置路径是绝对路径，优先使用 baseDir。
	// IsAbs 守卫不能省：filepath.Join 遇到绝对的第二参数不会丢弃 baseDir，
	// 而是静默拼接（Join("/app", "/etc/f.toml") == "/app/etc/f.toml"），指向错误文件。
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(baseDir, configPath)
	}

	var err error
	conf, err = config.LoadConfig(configPath)
	if err != nil {
		showError("配置加载失败", fmt.Sprintf("failed to load config: %v\n\n路径: %s", err, configPath))
		return
	}

	// 界面状态独立存于同一目录，不碰 config.toml
	statePath = filepath.Join(baseDir, "state.json")
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
	if err != nil {
		showError("托盘创建失败", fmt.Sprintf("failed to create tray: %v", err))
		return
	}
	tr.SetTooltip("TinyProxyAPI")

	// 建立初始勾选状态（NewTray 拿不到状态值，故在此设定一次）
	tr.SyncMenuState(logWin.IsVisible(), st.ShowLogOnStartup)

	// 关闭按钮和最小化都会在 tray 包内部隐藏日志窗口，main.go 只能通过
	// 这个回调感知到。回调触发时窗口已经隐藏，所以按当前可见性同步即可，
	// 无需判断是哪条路径。
	logWin.SetOnHidden(func() {
		tr.SyncMenuState(logWin.IsVisible(), st.ShowLogOnStartup)
	})

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
