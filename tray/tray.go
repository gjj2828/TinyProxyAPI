// tray/tray.go
package tray

import (
	"log"
	"time"

	"github.com/lxn/walk"
)

// Handlers 是托盘菜单各项的回调。tray 包只负责"显示菜单"和"报告菜单被点击"，
// 一次点击意味着什么（读写状态、显示窗口、退出）由调用方在 main.go 里决定。
type Handlers struct {
	OnToggleLog     func()
	OnToggleStartup func()
	OnExit          func()
}

type Tray struct {
	trayIcon      *walk.NotifyIcon
	app           *walk.Application
	mainWindow    *walk.MainWindow
	actShowLog    *walk.Action
	actStartup    *walk.Action
	onDoubleClick func()
	lastClick     time.Time
}

func NewTray(app *walk.Application, h Handlers) (*Tray, error) {
	t := &Tray{app: app}

	// 任一失败路径都走同一份清理：已创建的窗口/图标必须释放，否则残留隐藏的
	// MainWindow 与 NotifyIcon（菜单一旦 Add 过就泄漏得更完整）。成功返回时 err==nil，
	// defer 不会释放——所有权移交给调用方。
	var err error
	defer func() {
		if err != nil {
			if t.trayIcon != nil {
				t.trayIcon.Dispose()
			}
			if t.mainWindow != nil {
				t.mainWindow.Dispose()
			}
		}
	}()

	// Create a hidden main window as the host for NotifyIcon
	mw, err := walk.NewMainWindow()
	if err != nil {
		return nil, err
	}
	mw.SetVisible(false)
	t.mainWindow = mw

	ni, err := walk.NewNotifyIcon(mw)
	if err != nil {
		return nil, err
	}
	t.trayIcon = ni

	// 图标：直接取系统图标，失败只记录警告（内嵌资源降级链由后续任务补齐）。
	// 注意：系统图标取不到只是降级，不算失败，err 保持 nil 以免触发 defer 清理。
	var icon *walk.Icon
	icon, err = walk.NewIconFromSysDLL("shell32", 15)
	if err != nil {
		log.Printf("warn: system icon unavailable, tray may be invisible: %v", err)
		icon, err = nil, nil
	} else if err := t.trayIcon.SetIcon(icon); err != nil {
		return nil, err
	}

	// 必须显式调用 SetVisible(true) 才能显示托盘图标
	if err := t.trayIcon.SetVisible(true); err != nil {
		return nil, err
	}

	// 菜单只构建一次，此后仅通过 SyncMenuState 翻动勾选状态
	menu := t.trayIcon.ContextMenu()

	actShowLog := walk.NewAction()
	actShowLog.SetText("显示日志")
	actShowLog.SetCheckable(true)
	if h.OnToggleLog != nil {
		actShowLog.Triggered().Attach(h.OnToggleLog)
	}
	t.actShowLog = actShowLog

	actStartup := walk.NewAction()
	actStartup.SetText("日志启动时显示")
	actStartup.SetCheckable(true)
	if h.OnToggleStartup != nil {
		actStartup.Triggered().Attach(h.OnToggleStartup)
	}
	t.actStartup = actStartup

	// "退出"不设 Checkable：它是一次性动作，不该带勾选框
	actExit := walk.NewAction()
	actExit.SetText("退出")
	if h.OnExit != nil {
		actExit.Triggered().Attach(h.OnExit)
	}

	if err := menu.Actions().Add(actShowLog); err != nil {
		return nil, err
	}
	if err := menu.Actions().Add(actStartup); err != nil {
		return nil, err
	}
	if err := menu.Actions().Add(actExit); err != nil {
		return nil, err
	}

	return t, nil
}

func (t *Tray) SetTooltip(text string) {
	t.trayIcon.SetToolTip(text)
}

// SyncMenuState 把当前界面状态同步到菜单勾选框。仅设置勾选值，不改菜单结构。
func (t *Tray) SyncMenuState(showLog, startup bool) {
	t.actShowLog.SetChecked(showLog)
	t.actStartup.SetChecked(startup)
}

func (t *Tray) OnDoubleClick(handler func()) {
	t.onDoubleClick = handler
	t.trayIcon.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton {
			now := time.Now()
			if now.Sub(t.lastClick) < 300*time.Millisecond {
				if t.onDoubleClick != nil {
					t.onDoubleClick()
				}
			}
			t.lastClick = now
		}
	})
}

func (t *Tray) Destroy() {
	t.trayIcon.Dispose()
	if t.mainWindow != nil {
		t.mainWindow.Dispose()
	}
}

// MainWindow returns the hidden main window used as tray host
func (t *Tray) MainWindow() *walk.MainWindow {
	return t.mainWindow
}
