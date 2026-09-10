## Task 4: 创建日志窗口（tray/logwindow.go）

**Files:**
- Create: `tray/logwindow.go`

**Interfaces:**
- Consumes: 父窗口 `walk.MainWindow`
- Produces:
  - `LogWindow` struct
  - `ShowLogWindow(parent walk.MainWindow) (*LogWindow, error)`
  - `(*LogWindow).AppendText(text string)` — 追加日志文本
  - `(*LogWindow).IsVisible() bool`
  - `(*LogWindow).Show() / Hide()`

- [ ] **Step 1: 实现 LogWindow 结构体**

```go
// tray/logwindow.go
package tray

import (
	"github.com/lxn/walk"
	"github.com/lxn/win"
)

type LogWindow struct {
	*walk.MainWindow
	textEdit *walk.TextEdit
}

func ShowLogWindow(parent walk.MainWindow) (*LogWindow, error) {
	ml, err := walk.NewMainWindow(parent)
	if err != nil {
		return nil, err
	}

	ml.SetTitle("TinyProxyAPI - 日志")
	ml.SetWidth(600)
	ml.SetHeight(400)
	ml.SetLayout(walk.NewVBoxLayout())
	ml.Center()

	te, err := walk.NewTextEdit(ml)
	if err != nil {
		return nil, err
	}
	te.SetReadOnly(true)
	te.SetVerticalScrollBarPolicy(walk.ScrollBarAlwaysOn)

	ml.SetSize(600, 400)

	lw := &LogWindow{
		MainWindow: ml,
		textEdit:   te,
	}

	ml.Closing().Attach(func(cancel *bool) {
		*cancel = true
		lw.Hide()
	})

	return lw, nil
}

func (lw *LogWindow) AppendText(text string) {
	walk.NewGoroutineProperty().Enqueue(func() {
		lw.textEdit.AppendText(text)
	})
}

func (lw *LogWindow) IsVisible() bool {
	return lw.Visible()
}

func (lw *LogWindow) Show() {
	walk.NewGoroutineProperty().Enqueue(func() {
		lw.Show()
		lw.Activate()
	})
}

func (lw *LogWindow) Hide() {
	walk.NewGoroutineProperty().Enqueue(func() {
		lw.Hide()
	})
}
```

- [ ] **Step 2: 提交**

```bash
git add tray/logwindow.go
git commit -m "feat: add log window module"
```

---

