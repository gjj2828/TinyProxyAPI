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
		lw.Hide()
	})

	// 最小化时隐藏窗口（仅在托盘显示）
	// 定期检测窗口是否最小化，如果是则隐藏
	if hwnd != 0 {
		go func() {
			ticker := time.NewTicker(500 * time.Millisecond)
			defer ticker.Stop()
			for range ticker.C {
				if win.IsIconic(hwnd) && lw.Visible() {
					lw.Hide()
				}
			}
		}()
	}

	return lw, nil
}

func (lw *LogWindow) AppendText(text string) {
	lw.textEdit.AppendText(text)
}

func (lw *LogWindow) IsVisible() bool {
	return lw.Visible()
}

func (lw *LogWindow) Show() {
	lw.MainWindow.Show()
	lw.Activate()
}

func (lw *LogWindow) Hide() {
	lw.MainWindow.Hide()
}
