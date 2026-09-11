# 配置文件自动生成与托盘引导实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 当配置文件 `config.toml` 不存在时，自动基于内嵌模板创建初始配置文件，启动代理服务并弹出引导提示框，自动唤起编辑器打开配置文件；托盘右键菜单增加「打开配置文件」项；同步更新文档。

**Architecture:** 
1. 在 `config` 包中实现 `EnsureConfigFile(path string, fallbackTemplate []byte) (bool, error)`，并编写对应的单元测试；
2. 在 `tray` 包的 `Handlers` 中新增 `OnOpenConfig func()` 回调，并在托盘右键菜单顶部新增「打开配置文件」项与分隔线；
3. 在 `main.go` 中通过 `//go:embed config.toml.example` 嵌入默认模板，实现 `openConfigFile(path string)` 唤起逻辑，编排首次生成检测、自动打开与引导弹窗；
4. 同步更新 `README.md` 与 `README_EN.md`。

**Tech Stack:** Go 1.22+, `//go:embed`, `github.com/lxn/walk`, `github.com/lxn/win`, `exec.Command`

---

## Task 1: config 包 — 实现 EnsureConfigFile 与单元测试

**Files:**
- Create: `config/template.go`
- Create: `config/template_test.go`

**Interfaces:**
- Produces: `config.EnsureConfigFile(path string, fallbackTemplate []byte) (bool, error)`

- [ ] **Step 1: 编写 config/template_test.go 测试用例**

编写单元测试，覆盖以下场景：
1. 目标文件不存在时，`EnsureConfigFile` 成功创建文件，写入 fallbackTemplate，并返回 `created == true`；
2. 目标文件不存在且同目录下存在物理 `config.toml.example` 时，优先使用物理文件内容创建；
3. 目标文件已存在时，`EnsureConfigFile` 不修改文件内容，并返回 `created == false`。

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureConfigFile_NotExists(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "config.toml")
	template := []byte("# default template")

	created, err := EnsureConfigFile(targetPath, template)
	if err != nil {
		t.Fatalf("EnsureConfigFile 失败: %v", err)
	}
	if !created {
		t.Errorf("期望 created 为 true，实际为 false")
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("读取新创建文件失败: %v", err)
	}
	if string(content) != string(template) {
		t.Errorf("期望文件内容 %q，实际 %q", string(template), string(content))
	}
}

func TestEnsureConfigFile_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "config.toml")
	existingContent := []byte("# existing config")
	if err := os.WriteFile(targetPath, existingContent, 0644); err != nil {
		t.Fatalf("写入初始文件失败: %v", err)
	}

	template := []byte("# default template")
	created, err := EnsureConfigFile(targetPath, template)
	if err != nil {
		t.Fatalf("EnsureConfigFile 失败: %v", err)
	}
	if created {
		t.Errorf("期望 created 为 false，实际为 true")
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("读取文件失败: %v", err)
	}
	if string(content) != string(existingContent) {
		t.Errorf("原有文件内容被篡改，期望 %q，实际 %q", string(existingContent), string(content))
	}
}

func TestEnsureConfigFile_PrefersDiskExample(t *testing.T) {
	dir := t.TempDir()
	targetPath := filepath.Join(dir, "config.toml")
	examplePath := filepath.Join(dir, "config.toml.example")
	diskContent := []byte("# disk example content")
	if err := os.WriteFile(examplePath, diskContent, 0644); err != nil {
		t.Fatalf("写入 example 文件失败: %v", err)
	}

	fallbackTemplate := []byte("# fallback template")
	created, err := EnsureConfigFile(targetPath, fallbackTemplate)
	if err != nil {
		t.Fatalf("EnsureConfigFile 失败: %v", err)
	}
	if !created {
		t.Errorf("期望 created 为 true，实际为 false")
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("读取新创建文件失败: %v", err)
	}
	if string(content) != string(diskContent) {
		t.Errorf("期望优先使用同目录 example 内容 %q，实际 %q", string(diskContent), string(content))
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `go test ./config -run "TestEnsureConfigFile"`
Expected: FAIL (未定义 `EnsureConfigFile`)

- [ ] **Step 3: 创建 config/template.go 并实现 EnsureConfigFile**

```go
package config

import (
	"os"
	"path/filepath"
)

// EnsureConfigFile 检查目标路径配置文件是否存在；若不存在则自动从同目录 example 或 fallbackTemplate 生成。
// 返回 created 标识是否为本次新创建。
func EnsureConfigFile(path string, fallbackTemplate []byte) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}

	var content []byte
	examplePath := filepath.Join(filepath.Dir(path), "config.toml.example")
	if data, err := os.ReadFile(examplePath); err == nil && len(data) > 0 {
		content = data
	} else {
		content = fallbackTemplate
	}

	if err := os.WriteFile(path, content, 0644); err != nil {
		return false, err
	}
	return true, nil
}
```

- [ ] **Step 4: 运行测试验证通过**

Run: `go test ./config/...`
Expected: PASS

- [ ] **Step 5: 提交 Task 1 代码**

```bash
git add config/template.go config/template_test.go
git commit -m "feat(config): add EnsureConfigFile with fallback template support"
```

---

## Task 2: tray 包 — 扩展 Handlers 与右键菜单「打开配置文件」

**Files:**
- Modify: `tray/tray.go`

**Interfaces:**
- Updates: `tray.Handlers` 新增 `OnOpenConfig func()`
- Updates: `tray.NewTray` 在上下文菜单顶部增加「打开配置文件」与 `walk.NewSeparatorAction()`

- [ ] **Step 1: 修改 tray/tray.go**

在 `Handlers` 中新增 `OnOpenConfig func()`：
```go
type Handlers struct {
	OnOpenConfig    func()
	OnToggleLog     func()
	OnLeftClick     func()
	OnToggleStartup func()
	OnExit          func()
}
```

在 `NewTray` 中构建菜单动作时，在最顶部增加「打开配置文件」和分隔线：
```go
	menu := t.trayIcon.ContextMenu()

	actOpenConfig := walk.NewAction()
	actOpenConfig.SetText("打开配置文件")
	if h.OnOpenConfig != nil {
		actOpenConfig.Triggered().Attach(h.OnOpenConfig)
	}

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

	if err := menu.Actions().Add(actOpenConfig); err != nil {
		return nil, err
	}
	if err := menu.Actions().Add(walk.NewSeparatorAction()); err != nil {
		return nil, err
	}
	if err := menu.Actions().Add(actShowLog); err != nil {
		return nil, err
	}
	if err := menu.Actions().Add(actStartup); err != nil {
		return nil, err
	}
	if err := menu.Actions().Add(walk.NewSeparatorAction()); err != nil {
		return nil, err
	}
	if err := menu.Actions().Add(actExit); err != nil {
		return nil, err
	}
```

- [ ] **Step 2: 编译测试**

Run: `go test ./tray/...`
Expected: PASS

- [ ] **Step 3: 提交 Task 2 代码**

```bash
git add tray/tray.go
git commit -m "feat(tray): add open config menu item and separators"
```

---

## Task 3: main.go — 模板内嵌、编辑器唤起与启动引导流程编排

**Files:**
- Modify: `main.go`

- [ ] **Step 1: 在 main.go 中添加模板内嵌与 openConfigFile 辅助函数**

在 `main.go` 中添加：
```go
import (
	_ "embed"
	"os/exec"
	"syscall"
	// ... 现有引用保持不变
)

//go:embed config.toml.example
var defaultConfigTemplate []byte

// openConfigFile 使用 Windows 关联程序打开配置文件；若无关联程序则兜底使用记事本 notepad.exe。
func openConfigFile(path string) {
	verbPtr, _ := syscall.UTF16PtrFromString("open")
	filePtr, _ := syscall.UTF16PtrFromString(path)

	// win.ShellExecute 返回 HINSTANCE。在 Windows API 中，值 <= 32 表示失败（如 31=SE_ERR_NOASSOC）。
	ret := win.ShellExecute(0, verbPtr, filePtr, nil, nil, win.SW_SHOWNORMAL)
	if ret <= 32 {
		log.Printf("warn: ShellExecute failed (%d), falling back to notepad: %s", ret, path)
		cmd := exec.Command("notepad.exe", path)
		if err := cmd.Start(); err != nil {
			log.Printf("warn: failed to start notepad for config: %v", err)
		}
	}
}
```

- [ ] **Step 2: 在 main.go 的 main() 流程中编排 EnsureConfigFile 与引导提示**

在 `configPath` 计算完成后：
```go
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
```

在 `tray.NewTray` 中提供 `OnOpenConfig`：
```go
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
```

在启动后台 HTTP 服务与热重载监听之后、`tr.MainWindow().Run()` 之前添加引导提示：
```go
	if createdNewConfig {
		openConfigFile(configPath)
		walk.MsgBox(
			tr.MainWindow(),
			"提示",
			fmt.Sprintf("未检测到配置文件，已为您自动生成默认配置文件 config.toml。\n\n代理服务已在系统托盘运行，并已为您打开配置文件，请填写您的 API Key 与模型配置后保存，保存后将自动热重载生效。\n\n配置文件路径:\n%s", configPath),
			walk.MsgBoxIconInformation,
		)
	}

	// 运行消息循环
	tr.MainWindow().Run()
```

- [ ] **Step 3: 运行完整构建并测试编译**

Run: `.\build.ps1`
Expected: 编译成功生成 `TinyProxyAPI.exe`，无语法或链接错误。

- [ ] **Step 4: 提交 Task 3 代码**

```bash
git add main.go
git commit -m "feat: embed default config and auto-create with tray guidance"
```

---

## Task 4: 更新文档与端到端验证

**Files:**
- Modify: `README.md`
- Modify: `README_EN.md`

- [ ] **Step 1: 更新 README.md 与 README_EN.md**

在特性列表中增加：
- 自动生成配置：当 `config.toml` 缺失时自动创建并弹出引导
- 托盘菜单新增「打开配置文件」菜单项与分隔线说明

- [ ] **Step 2: 执行端到端验证**

1. 临时重命名 `config.toml` 为 `config.toml.bak`。
2. 启动 `.\TinyProxyAPI.exe`。
3. 验证：
   - 自动生成了 `config.toml`；
   - 自动拉起记事本或编辑器打开了 `config.toml`；
   - 弹出了包含正确路径的信息框；
   - 点击“确定”后，托盘图标正常显示，HTTP 服务正常运行；
   - 右键托盘图标，菜单包含「打开配置文件」且点击可再次唤起编辑器；
   - 保存修改后，日志窗口显示热重载生效。
4. 恢复原有 `config.toml`。

- [ ] **Step 3: 提交 Task 4 文档**

```bash
git add README.md README_EN.md
git commit -m "docs: update readme with auto-create config and tray menu item"
```
