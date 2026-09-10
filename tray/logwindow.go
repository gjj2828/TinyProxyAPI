// tray/logwindow.go
package tray

import (
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/win"
)

type LogWindow struct {
	*walk.MainWindow
	textEdit *walk.TextEdit
	// onHidden 在窗口"自我隐藏"时回调。关闭按钮的隐藏发生在本包内部的
	// Closing 处理里，main.go 无从察觉；不回调的话托盘菜单的"显示日志"
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

	// 最小化不做处理：走 Windows 原生最小化，任务栏保留图标。
	//
	// 之前这里有个 500ms 轮询，最小化后把窗口 SW_HIDE 掉（只留托盘）。改成不处理
	// 有两层原因：
	//  1. 语义上最小化与关闭的"隐藏"重复了，用户点最小化预期是看到任务栏图标；
	//  2. SW_HIDE 会污染 walk 内部的 proposedSize，还原时窗口被缩成 276x77
	//     （见 Show 的注释）。不 SW_HIDE 就没有这个污染，也不再有轮询 goroutine
	//     从后台线程调 walk 的 SetVisible。

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
// 窗口处于最小化（iconic）状态时必须先还原，不能直接调 walk 的 Show：
// walk 的 Show 内部走 SW_SHOWNA（见 walk/window.go 的 setWindowVisible），
// 而 SW_SHOWNA 不还原最小化窗口，窗口只会以任务栏图标形态"显示"——用户视角就是
// 点了"显示日志"但窗口不出来。
//
// 还原的顺序有讲究，三步都不能少：
//  1. 先从 RcNormalPosition 记下正常矩形；
//  2. SW_RESTORE 还原；
//  3. 用 SetBoundsPixels 把矩形设回去。
//
// 第 3 步是必需的：SW_RESTORE 期间 walk 的 startLayout 会按当时的 proposedSize
// 重新布局（form.go:681 里 cs < min 时调 SetSizePixels），而 proposedSize 在最小化
// 时被 WM_WINDOWPOSCHANGED 写成了任务栏图标的 276x45，窗口就会被缩成 276x77。
// SetBoundsPixels 除了设矩形，还会把 proposedSize 一并修正（form.go:217），
// 之后紧跟的 walk Show 里的重算才拿到正确尺寸。
func (lw *LogWindow) Show() {
	hwnd := lw.Handle()
	if hwnd != 0 && win.IsIconic(hwnd) {
		var wp win.WINDOWPLACEMENT
		wp.Length = uint32(unsafe.Sizeof(wp))
		var normal win.RECT
		if win.GetWindowPlacement(hwnd, &wp) {
			normal = wp.RcNormalPosition
		}

		win.ShowWindow(hwnd, win.SW_RESTORE)

		lw.SetBoundsPixels(walk.Rectangle{
			X:      int(normal.Left),
			Y:      int(normal.Top),
			Width:  int(normal.Right - normal.Left),
			Height: int(normal.Bottom - normal.Top),
		})
	}
	lw.MainWindow.Show()
	lw.Activate()
}

// Hide 隐藏窗口。菜单勾选框由调用方同步，因此这里不触发 onHidden。
func (lw *LogWindow) Hide() {
	lw.MainWindow.Hide()
}

// hideSelf 供窗口内部的自我隐藏路径（关闭按钮）使用：
// 隐藏并回调通知，让 main.go 能同步托盘菜单勾选框。
func (lw *LogWindow) hideSelf() {
	lw.MainWindow.Hide()
	if lw.onHidden != nil {
		lw.onHidden()
	}
}

// SetOnHidden 注册窗口被自身机制隐藏（关闭按钮）时的回调。
func (lw *LogWindow) SetOnHidden(fn func()) {
	lw.onHidden = fn
}
