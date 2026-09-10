// tray/tray.go
package tray

import (
	"log"
	"time"

	"github.com/lxn/walk"
)

// iconResourceID 是嵌入在可执行文件里的 GROUP_ICON 资源 ID。
//
// 该值是实测得来，不是推测。测量方式：go build 出 exe 后，用
// tools/resid 解析其 PE 资源目录（.rsrc 节），在 RT_GROUP_ICON(=14) 下
// 读到的 ID 即 2。同一份产物里 RT_MANIFEST=1，RT_ICON=3/4/5/6，
// GROUP_ICON 的组目录（62 字节）恰好引用了这 4 个图像条目。
// 重新生成 icon.ico / rsrc.syso 后可用 `go run ./tools/resid <exe>` 复核。
//
// 为什么用 GROUP_ICON 的 ID 而不是某个 RT_ICON 的 ID：walk.NewIconFromResourceId
// 走 newIconFromResource -> checkNewIcon -> handleForDPIWithError，最终调用
// Win32 LoadIconWithScaleDown(hInst, ...)，其中 hInst 是 exe 自身的实例句柄
// （见 walk/icon.go 的 NewIconFromResourceId 与 handleForDPIWithError）。
// LoadIcon 系按资源名
// 解析，图标组资源（GROUP_ICON）正是多尺寸 rc 图标的通用约定，所以传组 ID 即可，
// 系统会自行挑最合适的那一档尺寸。
const iconResourceID = 2

// loadAppIcon 取 exe 内嵌的应用图标，失败时逐级降级，永不返回错误。
//
//  1. 内嵌资源 iconResourceID（rsrc 打进 exe 的 GROUP_ICON）
//  2. shell32 第 15 号系统图标兜底——只为不留空白，不追求语义匹配
//  3. 再失败返回 nil，由调用方决定"无图标"如何呈现
//
// 返回 nil 不代表失败。图标取不到只是降级，不能把它当错误抛出去：否则 NewTray
// 里的 defer 会释放已创建的 MainWindow / NotifyIcon，把降级路径和失败路径的清理
// 搅在一起。托盘和日志窗口共用这一条链路，保证两处显示同一个图标。
func loadAppIcon() *walk.Icon {
	icon, err := walk.NewIconFromResourceId(iconResourceID)
	if err != nil {
		log.Printf("warn: embedded icon resource %d unavailable, falling back to system icon: %v",
			iconResourceID, err)
		fb, fbErr := walk.NewIconFromSysDLL("shell32", 15)
		if fbErr == nil {
			return fb
		}
		log.Printf("warn: system icon also unavailable, UI may show a generic icon: %v", fbErr)
		return nil
	}
	return icon
}

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

	// 图标取不到只是降级，loadAppIcon 已经把三级降级和警告日志收在内部；
	// 这里不需要给 err 赋值，否则会触发上面 defer 的清理，误伤已创建的
	// MainWindow / NotifyIcon。
	if appIcon := loadAppIcon(); appIcon != nil {
		if err := t.trayIcon.SetIcon(appIcon); err != nil {
			return nil, err
		}
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
