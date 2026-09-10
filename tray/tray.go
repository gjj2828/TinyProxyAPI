// tray/tray.go
package tray

import (
	"log"
	"os"
	"time"

	"github.com/lxn/walk"

	"github.com/user/TinyProxyAPI/config"
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
		// icon.ico 不存在时使用默认图标，继续运行
		log.Printf("warn: icon file not found, using default: %v", err)
	} else {
		if err := t.trayIcon.SetIcon(icon); err != nil {
			return nil, err
		}
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

// MainWindow returns the hidden main window used as tray host
func (t *Tray) MainWindow() *walk.MainWindow {
	return t.mainWindow
}

// UpdateMenu updates the tray menu with current state
func UpdateMenu(t *Tray, logWin *LogWindow, conf *config.Config, configPath string) {
	showLogChecked := logWin != nil && logWin.IsVisible()
	startupChecked := conf.UI.ShowLogOnStartup

	items := []MenuItem{
		{
			Text:    "显示日志",
			Checked: showLogChecked,
			OnClick: func() {
				ToggleLogWindow(logWin, t, conf, configPath)
			},
		},
		{
			Text:    "日志启动时显示",
			Checked: startupChecked,
			OnClick: func() {
				ToggleStartupSetting(logWin, t, conf, configPath)
			},
		},
		{
			Text: "退出",
			OnClick: func() {
				CleanupAndExit(logWin, t)
			},
		},
	}
	t.SetMenu(items)
}

// ToggleLogWindow toggles the log window visibility
func ToggleLogWindow(logWin *LogWindow, t *Tray, conf *config.Config, configPath string) {
	if logWin.IsVisible() {
		logWin.Hide()
	} else {
		logWin.Show()
	}
	UpdateMenu(t, logWin, conf, configPath)
}

// ToggleStartupSetting toggles the show log on startup setting
func ToggleStartupSetting(logWin *LogWindow, t *Tray, conf *config.Config, configPath string) {
	conf.UI.ShowLogOnStartup = !conf.UI.ShowLogOnStartup

	// 保存配置
	if err := config.SaveConfig(configPath, conf); err != nil {
		log.Printf("failed to save config: %v", err)
	}
	UpdateMenu(t, logWin, conf, configPath)
}

// CleanupAndExit cleans up resources and exits
func CleanupAndExit(logWin *LogWindow, t *Tray) {
	if t != nil {
		t.Destroy()
	}
	if logWin != nil {
		logWin.Hide()
	}
	os.Exit(0)
}
