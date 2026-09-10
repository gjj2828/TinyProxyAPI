# Tray Bugfix Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 修复托盘图标缺失、菜单无限增长、配置文件被重写三个缺陷，并完成状态与配置分离。

**Architecture:** 将托盘菜单改为一次性构建并通过回调驱动；将持久化状态迁移到独立 state.json；将图标改为内嵌资源并提供系统图标降级路径。tray 包仅负责 UI 展示，业务编排集中在 main.go。

**Tech Stack:** Go、walk、fsnotify、JSON、Windows 资源（rsrc）

**Spec:** docs/superpowers/specs/2026-09-10-tray-bugfix-design.md

## Global Constraints

- 图标资源必须内嵌到 rsrc.syso。
- rsrc.syso 必须保留现有 Common-Controls manifest。
- config.toml 必须完全只读。
- ShowLogOnStartup 默认值必须为 true。
- state.json 位于 exe 同目录。
- Save 使用临时文件 + rename 原子写入。
- tray 包不得依赖 config 包。
- 菜单结构只构建一次。
- 退出菜单不得带勾选框。

---

## File Structure

### 新增文件

- tools/genicon/main.go：生成 icon.ico
- state/state.go：状态读写实现
- state/state_test.go：状态测试
- icon.ico：生成后的图标资源

### 修改文件

- tray/tray.go：图标加载、菜单结构、Handlers、SyncMenuState
- main.go：托盘编排迁移、状态管理、路径获取
- config/config.go：删除 SaveConfig 与 UI 配置相关内容
- README.md：文档更新
- .gitignore：忽略 state.json
- rsrc.syso：更新资源文件

---

### Task 1: State 包

**Files:**

- Create: state/state.go
- Create: state/state_test.go

**Interfaces:**

- Produces:
  - type State struct { ShowLogOnStartup bool }
  - func Load(path string) *State
  - func Save(path string, s *State) error

- [ ] Step 1: 写状态测试

```go
func TestLoadDefaultWhenMissing(t *testing.T) {}
func TestLoadDefaultWhenInvalidJSON(t *testing.T) {}
func TestLoadFalseValue(t *testing.T) {}
func TestRoundTrip(t *testing.T) {}
func TestSaveLeavesNoTempFile(t *testing.T) {}
```

- [ ] Step 2: 运行测试确认失败

Run: `go test ./state`

Expected: FAIL，Load/Save 未定义。

- [ ] Step 3: 实现状态模型与默认值

```go
type State struct {
    ShowLogOnStartup bool `json:"show_log_on_startup"`
}
```

- [ ] Step 4: 实现 Load 与 Save

```go
func Load(path string) *State
func Save(path string, s *State) error
```

- [ ] Step 5: 运行测试确认通过

Run: `go test ./state`

Expected: PASS

### Task 2: Config 只读化

**Files:**

- Modify: config/config.go

**Interfaces:**

- Consumes: existing LoadConfig
- Produces: 不再存在 SaveConfig、UIConfig、Config.UI

- [ ] Step 1: 添加删除前失败编译检查

Run: `go test ./...`

Expected: 记录当前引用点。

- [ ] Step 2: 删除配置写回接口

```go
// 删除:
func SaveConfig(...)
type UIConfig struct
Config.UI
```

- [ ] Step 3: 修复编译错误引用

更新所有调用方。

- [ ] Step 4: 运行测试

Run: `go test ./config`

Expected: PASS

### Task 3: Tray 菜单重构

**Files:**

- Modify: tray/tray.go

**Interfaces:**

- Produces:
  - type Handlers struct
  - func NewTray(app *walk.Application, h Handlers) (*Tray, error)
  - func (t *Tray) SyncMenuState(showLog, startup bool)

- [ ] Step 1: 定义 Action 持有字段

```go
actShowLog *walk.Action
actStartup *walk.Action
```

- [ ] Step 2: 定义 Handlers

```go
type Handlers struct {
    OnToggleLog func()
    OnToggleStartup func()
    OnExit func()
}
```

- [ ] Step 3: 一次性创建菜单

创建显示日志、日志启动时显示、退出三项菜单。

- [ ] Step 4: 删除旧模型

删除：

```go
SetMenu
MenuItem
UpdateMenu
ToggleLogWindow
ToggleStartupSetting
CleanupAndExit
```

- [ ] Step 5: 实现状态同步

```go
func (t *Tray) SyncMenuState(showLog, startup bool)
```

### Task 4: 图标资源与加载

**Files:**

- Create: tools/genicon/main.go
- Modify: tray/tray.go
- Modify: rsrc.syso
- Create: icon.ico

**Interfaces:**

- Produces:
  - const iconResourceID
  - 资源加载降级链

- [ ] Step 1: 编写图标生成器

生成 16/32/48/256 尺寸 ICO。

- [ ] Step 2: 生成 icon.ico

Run:

```bash
go run ./tools/genicon
```

- [ ] Step 3: 生成资源文件

Run:

```bash
rsrc -manifest TinyProxyAPI.exe.manifest -ico icon.ico -o rsrc.syso
```

- [ ] Step 4: 实现三级降级

```go
walk.NewIconFromResourceId(...)
walk.NewIconFromSysDLL("shell32", 15)
```

- [ ] Step 5: 删除 iconPath 依赖

main.go 与 tray.go 同步调整。

### Task 5: Main 编排迁移

**Files:**

- Modify: main.go

**Interfaces:**

- Consumes:
  - state.Load
  - state.Save
  - tray.Handlers
  - tray.SyncMenuState

- [ ] Step 1: 获取 exe 目录

```go
exe, _ := os.Executable()
```

- [ ] Step 2: 加载 state.json

调用 state.Load。

- [ ] Step 3: 实现日志窗口切换逻辑

实现 OnToggleLog 回调。

- [ ] Step 4: 实现启动显示开关逻辑

修改内存状态并调用 state.Save。

- [ ] Step 5: 删除热重载菜单刷新

删除 UpdateMenu 调用链。

### Task 6: 文档与忽略规则

**Files:**

- Modify: README.md
- Modify: .gitignore

**Interfaces:**

- Produces: 更新后的使用说明

- [ ] Step 1: 忽略 state.json

新增：

```
state.json
```

- [ ] Step 2: 更新启动日志说明

说明状态保存在 state.json。

- [ ] Step 3: 更新图标说明

删除外部 icon.ico 要求。

- [ ] Step 4: 新增图标生成流程

记录 go run 与 rsrc 命令。

### Task 7: 验证与验收

**Files:**

- Test: state/state_test.go
- Test: tray 手工验证

**Interfaces:**

- Consumes: 全部前置任务

- [ ] Step 1: 运行状态测试

Run:

```bash
go test ./state
```

- [ ] Step 2: 运行全量测试

Run:

```bash
go test ./...
```

- [ ] Step 3: 手工验证托盘图标

确认有图标且无 walk 图标错误。

- [ ] Step 4: 手工验证菜单

连续点击 20 次仍保持三项。

- [ ] Step 5: 验证 config.toml 未变化

Run:

```powershell
certutil -hashfile config.toml SHA256
```

前后哈希一致。

## Self Review

- 缺陷 A 对应 Task 4。
- 缺陷 B 对应 Task 3 与 Task 5。
- 缺陷 C 对应 Task 1、Task 2、Task 5。
- README 更新对应 Task 6。
- 自动化测试与验收对应 Task 7。
- 无 TBD、TODO 或未定义占位项。

