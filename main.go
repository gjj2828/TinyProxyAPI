package main

import (
	_ "embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/gjj2828/TinyProxyAPI/config"
	"github.com/gjj2828/TinyProxyAPI/handler"
	"github.com/gjj2828/TinyProxyAPI/matcher"
	"github.com/gjj2828/TinyProxyAPI/signal"
	"github.com/gjj2828/TinyProxyAPI/state"
	"github.com/gjj2828/TinyProxyAPI/tray"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

//go:embed config.toml.example
var defaultConfigTemplate []byte

var (
	shell32          = syscall.NewLazyDLL("shell32.dll")
	procShellExecute = shell32.NewProc("ShellExecuteW")
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

// openConfigFile 使用 Windows 关联程序打开配置文件；若无关联程序则兜底使用记事本 notepad.exe。
func openConfigFile(path string) {
	fallbackNotepad := func() {
		cmd := exec.Command("notepad.exe", path)
		if err := cmd.Start(); err != nil {
			log.Printf("warn: failed to start notepad for config: %v", err)
		} else {
			_ = cmd.Process.Release()
		}
	}

	verbPtr, _ := syscall.UTF16PtrFromString("open")
	filePtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		log.Printf("warn: invalid config file path for ShellExecute: %v", err)
		fallbackNotepad()
		return
	}

	// ShellExecute 返回 HINSTANCE。在 Windows API 中，值 <= 32 表示失败（如 31=SE_ERR_NOASSOC）。
	// 注：lxn/win 的 ShellExecute 返回 bool (ret != 0)，无法按 <= 32 判断 SE_ERR_NOASSOC 等错误码，
	// 因此直接调用 ShellExecuteW 获取原始 HINSTANCE 进行判断。
	ret, _, _ := procShellExecute.Call(
		0,
		uintptr(unsafe.Pointer(verbPtr)),
		uintptr(unsafe.Pointer(filePtr)),
		0,
		0,
		uintptr(win.SW_SHOWNORMAL),
	)
	if ret <= 32 {
		log.Printf("warn: ShellExecute failed (%d), falling back to notepad: %s", ret, path)
		fallbackNotepad()
	}
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

// activateLogWindow 将日志窗口显示并激活到前台。
// 隐藏时调用 Show()（内部处理最小化还原）；已显示时用 SetForegroundWindow+
// BringWindowToTop 强制提到 Z-order 最前（walk.Activate 在托盘回调上下文可能
// 不够可靠）。
// 菜单勾选框由调用方同步。
func activateLogWindow() {
	if !logWin.IsVisible() {
		logWin.Show()
		tr.SyncMenuState(true, st.ShowLogOnStartup)
		return
	}

	// 已显示（未最小化）→ 直接强制前台。
	// walk.Activate 只走 SetActiveWindow，从托盘回调上下文触发时
	// SetForegroundWindow 才能保证把窗口提到 Z-order 最前；
	// BringWindowToTop 进一步防止 Z-order 顺序问题。
	if hwnd := win.HWND(logWin.Handle()); hwnd != 0 {
		win.SetForegroundWindow(hwnd)
		win.BringWindowToTop(hwnd)
	}
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

	createdNewConfig, ensureErr := config.EnsureConfigFile(configPath, defaultConfigTemplate)
	if ensureErr != nil {
		showError("配置文件创建失败", fmt.Sprintf("failed to create default config: %v\n\n路径: %s", ensureErr, configPath))
		return
	}

	var err error
	conf, err = config.LoadConfig(configPath)
	if err != nil {
		showError("配置加载失败", fmt.Sprintf("failed to load config: %v\n\n路径: %s", err, configPath))
		return
	}

	// 界面状态独立存于用户目录 ~/.tinyproxyapi/state.json，不碰 config.toml
	var stateErr error
	statePath, stateErr = state.DefaultPath()
	if stateErr != nil {
		log.Printf("warn: failed to resolve state path: %v", stateErr)
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
		OnOpenConfig: func() {
			openConfigFile(configPath)
		},
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
	if err != nil {
		showError("托盘创建失败", fmt.Sprintf("failed to create tray: %v", err))
		return
	}
	tr.SetTooltip("TinyProxyAPI")

	// 建立初始勾选状态（NewTray 拿不到状态值，故在此设定一次）
	tr.SyncMenuState(logWin.IsVisible(), st.ShowLogOnStartup)

	// 关闭按钮会在 tray 包内部隐藏日志窗口（tray 包拦截 Closing 改成 Hide），
	// main.go 只能通过这个回调感知到。回调触发时窗口已经隐藏，所以按当前可见性
	// 同步即可。最小化走 Windows 原生最小化、不隐藏，因此不经过这个回调——
	// 此时勾选框会保持已勾选，直到下一次菜单交互才刷新（walk 没有最小化事件，
	// 要即时更新只能加窗口过程钩子或轮询）。
	logWin.SetOnHidden(func() {
		tr.SyncMenuState(logWin.IsVisible(), st.ShowLogOnStartup)
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
			log.Println("config reloaded successfully")
		}
	}()

	if createdNewConfig {
		walk.MsgBox(
			tr.MainWindow(),
			"提示",
			fmt.Sprintf("未检测到配置文件，已为您自动生成默认配置文件 config.toml。\n\n代理服务已在系统托盘运行。点击“确定”后将自动为您打开配置文件，请填写您的 API Key 与模型配置后保存，保存后将自动热重载生效。\n\n配置文件路径:\n%s", configPath),
			walk.MsgBoxIconInformation,
		)
		openConfigFile(configPath)
	}

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
