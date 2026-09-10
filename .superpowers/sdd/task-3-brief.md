## Task 3: 创建托盘管理（tray/tray.go）

**Files:**
- Create: `tray/tray.go`

**Interfaces:**
- Consumes: `*walk.App`，托盘图标路径
- Produces:
  - `Tray` struct
  - `NewTray(app *walk.App, iconPath string) (*Tray, error)`
  - `SetTooltip(text string)`
  - `SetMenu(items []MenuItem)` — 设置菜单项
  - `OnDoubleClick(handler func())`
  - `Destroy()`

- [ ] **Step 1: 实现 Tray 结构体和 NewTray 函数**

```go
// tray/tray.go
package tray

import (
	"github.com/lxn/win"
	"github.com/lxn/walk"
)

type Tray struct {
	trayIcon       *walk.NotifyIcon
	app            *walk.App
	onDoubleClick  func()
}

type MenuItem struct {
	Text      string
	Checked   bool
	OnClick   func()
}

func NewTray(app *walk.App, iconPath string) (*Tray, error) {
	t := &Tray{app: app}

	ni, err := walk.NewNotifyIcon(t.app)
	if err != nil {
		return nil, err
	}
	t.trayIcon = ni

	if err := t.trayIcon.SetIcon(iconPath); err != nil {
		return nil, err
	}

	return t, nil
}

func (t *Tray) SetTooltip(text string) {
	t.trayIcon.SetToolTip(text)
}

func (t *Tray) SetMenu(items []MenuItem) {
	menu, _ := walk.NewMenu()
	for _, item := range items {
		action := walk.NewAction()
		action.SetText(item.Text)
		action.SetChecked(item.Checked)
		action.Clicked().Attach(item.OnClick)
		menu.Actions().Add(action)
	}
	t.trayIcon.SetContextMenu(menu)
}

func (t *Tray) OnDoubleClick(handler func()) {
	t.onDoubleClick = handler
	t.trayIcon.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button == walk.LeftButton() && t.onDoubleClick != nil {
			t.onDoubleClick()
		}
	})
}

func (t *Tray) Destroy() {
	t.trayIcon.Dispose()
}
```

- [ ] **Step 2: 提交**

```bash
git add tray/tray.go
git commit -m "feat: add tray management module"
```

---

