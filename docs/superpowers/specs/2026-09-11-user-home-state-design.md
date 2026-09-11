# state.json 存储迁移至用户目录设计

## 概述

将程序运行时维护的界面状态文件 `state.json` 从可执行文件同级目录迁移至用户家目录下的 `.tinyproxyapi` 文件夹（`~/.tinyproxyapi/state.json`），并在保存时自动创建所需父目录。

## 背景与动机

目前 `state.json` 默认存放于可执行文件所在目录：
1. **权限受限**：若将 `TinyProxyAPI.exe` 放置在如 `C:\Program Files\` 等系统保护目录，无管理员权限运行会导致状态写入失败。
2. **工作区污染**：在开发与本地测试时，会在项目根目录产生临时 `state.json`，存在污染 git 状态的隐患。
3. **版本更新不连贯**：替换二进制文件或从不同位置启动时，无法共享历史界面偏好（如「启动时显示日志」）。

将状态文件放置在用户家目录下（跨平台标准做法），可彻底解决上述问题。

## 行为规范

### 1. 存储路径规范
- **默认路径**：`~/.tinyproxyapi/state.json`
  - Windows: `%USERPROFILE%\.tinyproxyapi\state.json`（如 `C:\Users\<User>\.tinyproxyapi\state.json`）
  - Linux / macOS: `$HOME/.tinyproxyapi/state.json`
- **解析机制**：调用 Go 标准库 `os.UserHomeDir()` 获取家目录。
- **降级容错（Fallback）**：若 `os.UserHomeDir()` 返回错误（罕见的受限环境或缺少环境变量），回退至可执行文件同级目录；若仍不可用，回退至系统临时目录 `filepath.Join(os.TempDir(), ".tinyproxyapi", "state.json")`。始终返回可用的绝对路径。

### 2. 状态生命周期与过渡策略
- **平滑独立初始化**：不探查或迁移 exe 目录下的旧文件。若 `~/.tinyproxyapi/state.json` 不存在，`state.Load()` 直接使用内存默认值（`ShowLogOnStartup: true`），不产生启动副作用。
- **按需落盘与父目录自动创建**：仅在用户通过托盘菜单修改设置触发 `state.Save()` 时落盘。保存前自动调用 `os.MkdirAll(filepath.Dir(path), 0755)` 创建 `.tinyproxyapi` 目录。
- **原子覆盖**：写入同目录下 `.tmp` 临时文件后通过 `os.Rename` 原子覆盖目标文件，避免崩溃导致文件截断。

## 代码变更

### 1. state/state.go

**新增 `DefaultPath` 函数**：
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
```

**更新 `Save` 函数**：在写临时文件前确保父目录存在。
```go
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

### 2. main.go

将原本直接拼接 `filepath.Join(baseDir, "state.json")` 改为调用 `state.DefaultPath()`：

```go
// 界面状态独立存于用户目录 ~/.tinyproxyapi/state.json，不碰 config.toml
var stateErr error
statePath, stateErr = state.DefaultPath()
if stateErr != nil {
    log.Printf("warn: failed to resolve state path: %v", stateErr)
}
st = state.Load(statePath)
```

### 3. README.md & README_EN.md

更新托盘与状态说明部分：
- `README.md`：更新为保存在用户目录 `~/.tinyproxyapi/state.json` 中。
- `README_EN.md`：更新为 saved in `~/.tinyproxyapi/state.json` in the user's home directory。

## 测试策略

### 1. 单元测试 (`state/state_test.go`)
- **`TestDefaultPath`**：
  - 验证返回路径非空且为绝对路径；
  - 验证路径以 `filepath.Join(".tinyproxyapi", "state.json")` 结尾。
- **`TestSave_CreatesDirectory`**：
  - 在空的临时目录下，传入尚未存在的嵌套路径（如 `filepath.Join(t.TempDir(), "nested", "state.json")`）；
  - 调用 `Save`，验证父目录被自动创建，且文件正常写入。
- **既有用例回归**：
  - 确保默认值、正常加载、损坏 JSON 恢复、原子写入等原有测试继续全部通过。

### 2. 回归测试
- 运行 `go test ./...` 确保所有模块测试通过。
- 编译并在本地验证构建无误。
