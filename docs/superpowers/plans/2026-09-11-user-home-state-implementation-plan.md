# state.json 存储迁移至用户目录实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将界面状态文件 `state.json` 迁移到用户家目录 `~/.tinyproxyapi/state.json`，在保存时自动创建父级目录，并更新主程序编排与文档。

**Architecture:** 在 `state` 包中新增 `DefaultPath() (string, error)` 函数解析 `~/.tinyproxyapi/state.json` 并提供安全降级（exe 目录、临时目录）；更新 `state.Save` 在写入前执行 `os.MkdirAll`；在 `main.go` 中通过 `state.DefaultPath()` 获取状态路径；同步更新 `README.md` 与 `README_EN.md`。

**Tech Stack:** Go 1.22+, `os.UserHomeDir`, `filepath.Join`, `testing`

---

## Task 1: state 包 — 新增 DefaultPath 与自动创建父目录

**Files:**
- Modify: `state/state.go`
- Modify: `state/state_test.go`

**Interfaces:**
- Produces: `state.DefaultPath() (string, error)`
- Updates: `state.Save(path string, s *State) error` 支持自动创建父目录

- [ ] **Step 1: 在 state/state_test.go 中添加针对 DefaultPath 与 Save 自动建目录的失败测试**

在 `state/state_test.go` 末尾添加如下测试函数：

```go
// 验证 DefaultPath 返回非空绝对路径且以 .tinyproxyapi/state.json 结尾。
func TestDefaultPath(t *testing.T) {
	p, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath 返回错误: %v", err)
	}
	if !filepath.IsAbs(p) {
		t.Errorf("DefaultPath 应返回绝对路径，实际为: %s", p)
	}

	expectedSuffix := filepath.Join(".tinyproxyapi", "state.json")
	if !filepath.IsAbs(p) || len(p) < len(expectedSuffix) || p[len(p)-len(expectedSuffix):] != expectedSuffix {
		t.Errorf("DefaultPath 路径应以 %s 结尾，实际为: %s", expectedSuffix, p)
	}
}

// 验证 Save 在父目录尚不存在时能自动创建父目录并成功写入。
func TestSave_CreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	nestedPath := filepath.Join(dir, ".tinyproxyapi", "state.json")

	if err := Save(nestedPath, &State{ShowLogOnStartup: false}); err != nil {
		t.Fatalf("Save 自动创建目录失败: %v", err)
	}

	loaded := Load(nestedPath)
	if loaded.ShowLogOnStartup {
		t.Errorf("从新创建目录加载状态失败，期望 false，实际 %v", loaded.ShowLogOnStartup)
	}
}
```

- [ ] **Step 2: 运行测试以验证测试因未实现而失败**

Run: `go test ./state -run "TestDefaultPath|TestSave_CreatesDirectory"`
Expected: FAIL (编译报错或未定义 `DefaultPath`)

- [ ] **Step 3: 修改 state/state.go 实现 DefaultPath 与 Save 自动创建父目录**

在 `state/state.go` 中新增 `DefaultPath()` 函数，并在 `Save` 函数起始处添加 `os.MkdirAll`：

```go
// DefaultPath 返回状态文件的默认存储路径 ~/.tinyproxyapi/state.json。
// 若获取用户家目录失败，将优雅回退至可执行文件所在目录或系统临时目录。
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".tinyproxyapi", "state.json"), nil
	}

	// 降级策略：尝试可执行文件所在目录
	if exe, exeErr := os.Executable(); exeErr == nil {
		return filepath.Join(filepath.Dir(exe), "state.json"), nil
	}

	// 最终兜底：系统临时目录
	return filepath.Join(os.TempDir(), ".tinyproxyapi", "state.json"), nil
}

// Save 以原子方式写入状态文件。
//
// 先确保父目录存在，然后写同目录下的临时文件，再 Rename 覆盖目标，
// 保证写入中途崩溃不会留下被截断的 JSON。成功后不留下任何临时文件。
func Save(path string, s *State) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.Marshal(s)
	if err != nil {
		return err
	}

	tmp := filepath.Join(dir, filepath.Base(path)+".tmp")
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}

	return nil
}
```

- [ ] **Step 4: 运行 state 包所有单元测试验证全部通过**

Run: `go test -v ./state`
Expected: PASS (包括原有用例与新增用例全部通过)

- [ ] **Step 5: 提交 state 包变更**

```bash
git add state/state.go state/state_test.go
git commit -m "feat(state): add DefaultPath and auto-create directory in Save"
```

---

## Task 2: main.go — 接入 state.DefaultPath

**Files:**
- Modify: `main.go:110-114`

**Interfaces:**
- Consumes: `state.DefaultPath() (string, error)`
- Consumes: `state.Load(statePath string) *State`

- [ ] **Step 1: 修改 main.go 中的状态路径获取逻辑**

在 `main.go` 中，找到原本硬编码拼接 `filepath.Join(baseDir, "state.json")` 的代码段：

```go
	// 界面状态独立存于同一目录，不碰 config.toml
	statePath = filepath.Join(baseDir, "state.json")
	st = state.Load(statePath)
```

替换为：

```go
	// 界面状态独立存于用户目录 ~/.tinyproxyapi/state.json，不碰 config.toml
	var stateErr error
	statePath, stateErr = state.DefaultPath()
	if stateErr != nil {
		log.Printf("warn: failed to resolve state path: %v", stateErr)
	}
	st = state.Load(statePath)
```

- [ ] **Step 2: 验证主程序编译正常**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 3: 提交 main.go 变更**

```bash
git add main.go
git commit -m "feat: use state.DefaultPath to store state in user directory"
```

---

## Task 3: 文档更新 (README.md 与 README_EN.md)

**Files:**
- Modify: `README.md`
- Modify: `README_EN.md`

- [ ] **Step 1: 修改 README.md 中的状态文件存储路径描述**

将 `README.md` 第 110 行：
```markdown
「日志启动时显示」的设置**不由 `config.toml` 管理**，而是由托盘菜单控制，保存在 exe 同目录的 `state.json` 中，默认值为启动时显示。
```
修改为：
```markdown
「日志启动时显示」的设置**不由 `config.toml` 管理**，而是由托盘菜单控制，保存在用户目录的 `~/.tinyproxyapi/state.json` 中，默认值为启动时显示。
```

- [ ] **Step 2: 修改 README_EN.md 中的状态文件存储路径描述**

将 `README_EN.md` 第 110 行：
```markdown
The "Show log on startup" setting is **not managed by `config.toml`**, but controlled via the tray menu and saved in `state.json` in the same directory as the exe, defaulting to show on startup.
```
修改为：
```markdown
The "Show log on startup" setting is **not managed by `config.toml`**, but controlled via the tray menu and saved in `~/.tinyproxyapi/state.json` in the user's home directory, defaulting to show on startup.
```

- [ ] **Step 3: 提交文档变更**

```bash
git add README.md README_EN.md
git commit -m "docs: update state.json path in README and README_EN"
```

---

## Task 4: 工作区清理与整体验证

**Files:**
- Delete: `state.json` (若项目根目录存在残留未跟踪文件)

- [ ] **Step 1: 清理根目录下遗留的 state.json 文件**

若项目根目录下存在遗留的 `state.json`，将其删除：
```powershell
Remove-Item -Path .\state.json -ErrorAction SilentlyContinue
```

- [ ] **Step 2: 执行全量测试**

Run: `go test -v ./...`
Expected: 全部测试包（`config`, `handler`, `matcher`, `state`, `tray`）均输出 PASS。

- [ ] **Step 3: 执行构建验证**

Run: `powershell -File .\build.ps1`
Expected: 成功构建 `TinyProxyAPI.exe`，退出码为 0。

- [ ] **Step 4: 检查 git 状态**

Run: `git status`
Expected: working tree clean。
