// tray/logwindow.go
package tray

import (
	"time"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

type LogWindow struct {
	*walk.MainWindow
	textEdit *walk.TextEdit
	// onHidden 在窗口"自我隐藏"时回调。关闭按钮和最小化轮询这两条隐藏路径
	// 都在本包内部触发，main.go 无从察觉；不回调的话托盘菜单的"显示日志"
	// 勾选框会停留在已勾选状态。
	onHidden func()
}

func ShowLogWindow() (*LogWindow, error) {
	ml, err := walk.NewMainWindow()
	if err != nil {
		return nil, err
	}

	ml.SetTitle("TinyProxyAPI - 日志")
	ml.SetWidth(600)
	ml.SetHeight(400)
	// 窗口图标必须显式设置：walk 的 MainWindow 创建流程不发 WM_SETICON，
	// WM_GETICON 保持 NULL 时标题栏左上角会退回 Windows 的通用窗口图标。
	// 与托盘共用 loadAppIcon，保证两处显示同一个图标。
	if appIcon := loadAppIcon(); appIcon != nil {
		if err := ml.SetIcon(appIcon); err != nil {
			ml.Dispose()
			return nil, err
		}
	}
	ml.SetLayout(walk.NewVBoxLayout())

	// Center window on screen using win32 API
	hwnd := ml.Handle()
	if hwnd != 0 {
		var rect win.RECT
		win.GetWindowRect(hwnd, &rect)
		screenWidth := win.GetSystemMetrics(win.SM_CXSCREEN)
		screenHeight := win.GetSystemMetrics(win.SM_CYSCREEN)
		winWidth := rect.Right - rect.Left
		winHeight := rect.Bottom - rect.Top
		win.SetWindowPos(hwnd, 0,
			(screenWidth-winWidth)/2,
			(screenHeight-winHeight)/2,
			0, 0,
			win.SWP_NOZORDER|win.SWP_NOSIZE)
	}

	te, err := walk.NewTextEdit(ml)
	if err != nil {
		return nil, err
	}
	te.SetReadOnly(true)

	lw := &LogWindow{
		MainWindow: ml,
		textEdit:   te,
	}

	ml.Closing().Attach(func(cancel *bool, reason walk.CloseReason) {
		*cancel = true
		lw.hideSelf()
	})

	// 最小化时隐藏窗口（仅在托盘显示）
	// 定期检测窗口是否最小化，如果是则隐藏
	if hwnd != 0 {
		go func() {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			for range ticker.C {
				if win.IsIconic(hwnd) && lw.Visible() {
					lw.hideSelf()
				}
			}
		}()
	}

	return lw, nil
}

func (lw *LogWindow) AppendText(text string) {
	lw.textEdit.AppendText(text)
}

// IsVisible 报告窗口是否处于"正常可见"状态。
//
// 最小化（iconic）的窗口 IsWindowVisible 仍返回 true，但对我们而言它等同于
// 未显示：用户看不到内容，托盘菜单勾选框也应取消。不在此排除 iconic 的话，
// 最小化后点"显示日志"会被判成"已可见"从而执行隐藏，而非把窗口还原出来。
func (lw *LogWindow) IsVisible() bool {
	if !lw.Visible() {
		return false
	}
	if hwnd := lw.Handle(); hwnd != 0 {
		return !win.IsIconic(hwnd)
	}
	return true
}

// Show 显示并激活窗口。
//
// 必须先解除 iconic 状态：walk 的 Show 内部走 SW_SHOWNA（见 walk/window.go 的
// setWindowVisible），而 SW_SHOWNA 对最小化窗口不会还原，窗口会以任务栏图标形态
// "显示"——用户视角就是点了"显示日志"但窗口不出来。最小化路径用 SW_HIDE 隐藏
// 窗口，SW_HIDE 不清除 iconic 标志，所以被最小化隐藏的窗口仍处于 iconic 状态。
// 若改在隐藏侧清理 iconic，需要先 SW_RESTORE 再 SW_HIDE，窗口会闪一下，故只在此处处理。
func (lw *LogWindow) Show() {
	if hwnd := lw.Handle(); hwnd != 0 && win.IsIconic(hwnd) {
		win.ShowWindow(hwnd, win.SW_RESTORE)
	}
	lw.MainWindow.Show()
	lw.Activate()
}

// Hide 隐藏窗口。菜单勾选框由调用方同步，因此这里不触发 onHidden。
func (lw *LogWindow) Hide() {
	lw.MainWindow.Hide()
}

// hideSelf 供窗口内部的自我隐藏路径（关闭按钮、最小化轮询）使用：
// 隐藏并回调通知，让 main.go 能同步托盘菜单勾选框。
func (lw *LogWindow) hideSelf() {
	lw.MainWindow.Hide()
	if lw.onHidden != nil {
		lw.onHidden()
	}
}

// SetOnHidden 注册窗口被自身机制隐藏（关闭按钮、最小化）时的回调。
func (lw *LogWindow) SetOnHidden(fn func()) {
	lw.onHidden = fn
}
