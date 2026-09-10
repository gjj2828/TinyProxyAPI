// tray/tray.go
package tray

import (
	"time"

	"github.com/lxn/walk"
)

type Tray struct {
	trayIcon     *walk.NotifyIcon
	app          *walk.Application
	mainWindow   *walk.MainWindow
	onDoubleClick func()
	lastClick    time.Time
}

type MenuItem struct {
	Text    string
	Checked bool
	OnClick func()
}

func NewTray(app *walk.Application, iconPath string) (*Tray, error) {
	t := &Tray{app: app}

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

	icon, err := walk.NewIconFromFile(iconPath)
	if err != nil {
		return nil, err
	}
	if err := t.trayIcon.SetIcon(icon); err != nil {
		return nil, err
	}

	return t, nil
}

func (t *Tray) SetTooltip(text string) {
	t.trayIcon.SetToolTip(text)
}

func (t *Tray) SetMenu(items []MenuItem) {
	menu := t.trayIcon.ContextMenu()
	for _, item := range items {
		action := walk.NewAction()
		action.SetText(item.Text)
		action.SetChecked(item.Checked)
		action.SetCheckable(true)
		action.Triggered().Attach(item.OnClick)
		menu.Actions().Add(action)
	}
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
