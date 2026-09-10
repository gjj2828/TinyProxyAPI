# TinyProxyAPI 托盘模块缺陷修复设计

日期：2026-09-10

## 背景

系统托盘功能上线后暴露三个缺陷：托盘区没有图标、菜单项每次点击后累积增长、程序在翻动开关时把用户手写的 `config.toml` 整体重写。三者都源自 `tray` 模块的同一条调用链，因此合并为一次修复。

## 缺陷根因

### 缺陷 A：托盘无图标，启动时打印 walk 错误堆栈

`tray/tray.go` 用 `walk.NewIconFromFile(iconPath)` 读取 exe 同目录的 `icon.ico`。该文件不存在，调用必然失败并打印 `LoadIconWithScaleDown` 堆栈。

失败分支只写了一行日志。代码注释声称"使用默认图标，继续运行"，但实际没有任何兜底逻辑——`NotifyIcon` 始终没拿到图标，托盘区因此是空白的。日志里那句警告和肉眼看到的"没有图标"是同一个原因的两种表现。

仓库里的 `rsrc.syso` 只有 676 字节，内容仅为 manifest，不含任何图标资源，因此 exe 内部也没有可回退的图标。

### 缺陷 B：菜单项每次点击累积三行

`Tray.SetMenu` 只调用 `menu.Actions().Add(action)`，从不清空既有菜单项。

`UpdateMenu` 会在四个时机被调用：启动初始化、点击"显示日志"、点击"日志启动时显示"、配置热重载。每次调用都向同一个 ContextMenu 追加三个新 Action，菜单因此单调增长。

附带问题：三个 Action 统一执行了 `SetCheckable(true)`，导致"退出"也带勾选框。

### 缺陷 C：配置文件被整体重写

`ToggleStartupSetting` 调用 `config.SaveConfig`，后者是 `toml.Marshal(cfg)` 加 `os.WriteFile`。

`Config` 结构体只有 `AllowedKeys`、`Platform.BaseURL`、`Platform.APIKey`、`UIConfig.ShowLogOnStartup` 带 toml tag，其余字段靠字段名匹配。序列化时按 Go 字段名输出，产生以下破坏：

- section 名变为大写：`[server]` → `[Server]`、`[models]` → `[Models]`
- 平台的内联表 `{ base_url = ..., api_key = ... }` 被拆成多行 `[Platforms.xxx]` 段
- 全部注释丢失，包括被注释掉的备用配置项
- 未设值的 `[default]` 段被写成 `Platform = ''`

Go 生态中没有 TOML 库能在序列化时保留注释，因此"换库"不构成解决方案。

两个次生问题：`os.WriteFile` 非原子写，而 fsnotify 正监听该文件，写入过程中被读取会触发一次失败重载；写入完成后 fsnotify 触发热重载，程序绕一圈重新加载自己刚写坏的文件。

## 设计决策

| 决策点 | 选择 | 理由 |
|---|---|---|
| 图标来源 | 代码生成 | 无现成素材；生成器留仓库便于后续调整 |
| 图标形态 | 分叉路由，翠绿 `#10b981` 线条，无实心底 | 一进三出对应按模型名分流的核心职责；笔画少，16px 下不糊 |
| 图标载体 | 内嵌 exe 资源（`rsrc.syso`） | 单文件分发不丢图标；exe 在资源管理器中也获得图标 |
| `.syso` 管理 | 生成产物提交进仓库 | 克隆后无需安装工具即可构建；生成命令写入 README |
| 配置写回 | 移除写回能力，状态另存 | 人工维护的配置与程序状态生命周期不同，混存是缺陷 C 的根源 |
| 状态文件格式 | JSON | 标准库支持；与 TOML 配置形成语义区分 |
| 菜单修复 | 一次性构建，仅同步勾选状态 | 菜单结构固定，增长缺陷从结构上不再可能发生 |

### 被否决的方案

**给结构体补 toml tag。** 只能修正 key 大小写，注释和内联表写法依旧丢失，不解决核心损失。

**就地改写 config.toml 单行。** 能完整保留注释与格式，但保留了"程序写用户配置文件"这一模式，且需处理段感知、行尾注释、key 缺失时插入等边界情况。移除写回能力更彻底。

**每次清空后重建菜单。** 仅需一行 `Actions().Clear()`，但把"无限增长"换成"反复销毁重建"，错误建模仍在，正确性依赖调用方记得先清空。

## 变更详情

### 1. 图标资源

**新增 `tools/genicon/`**：独立的一次性生成器，纯 Go 实现，不引入新依赖。按分叉路由几何绘制——三条贝塞尔分支加四个端点圆，颜色 `#10b981`——在 4 倍分辨率下渲染后降采样实现抗锯齿，输出 16、32、48、256 四个尺寸，打包为 ICO 容器（使用 Vista 以上支持的 PNG 内嵌格式）。产物 `icon.ico` 提交进仓库。

**生成 `rsrc.syso`**：

```
rsrc -manifest TinyProxyAPI.exe.manifest -ico icon.ico -o rsrc.syso
```

新 `.syso` 必须继续包含现有 manifest 中的 `Microsoft.Windows.Common-Controls` 依赖声明，walk 的控件渲染依赖它。生成产物提交进仓库，`go build` 自动链接，`build.ps1` 无需修改。

**修改 `tray.NewTray`**：删除 `iconPath` 参数，改为三级降级加载：

1. `walk.NewIconFromResourceId(iconResourceID)` 读取内嵌资源
2. 失败则 `walk.NewIconFromSysDLL("shell32", 15)` 取系统图标兜底
3. 再失败才记录警告

无论走哪条路径，托盘区都有可见图标。main.go 中拼接 `icon.ico` 路径的代码一并删除。

第二级的 `shell32.dll` 索引 15 只用于保证托盘不空白，不追求语义匹配。实现时目视确认该索引能正常渲染；若异常则改用索引 2，并在代码注释中记录替换原因。

**资源 ID 确定方式**：`rsrc` 分配的 GROUP_ICON 资源 ID 取决于 ICO 内的图像数量，非固定值。实现时须从生成的 `.syso` 中读取实际 ID，写为常量 `iconResourceID`。不得凭推测填写，也不得在运行时遍历试探。

### 2. 配置只读化与状态分离

**删除**：`config.SaveConfig` 函数、`Config.UI` 字段、`UIConfig` 类型。删除后代码中不存在任何写入 `config.toml` 的路径。

**新增 `state` 包**：

```go
type State struct {
    ShowLogOnStartup bool `json:"show_log_on_startup"`
}

func Load(path string) *State
func Save(path string, s *State) error
```

`Load` 不返回 error。文件缺失、内容损坏、权限不足一律返回默认值继续运行。这与 `config.LoadConfig` 的失败即退出形成有意对比：缺少配置无法工作，缺少状态只影响日志窗口是否自动弹出。

`ShowLogOnStartup` 默认值为 `true`。由于 Go 布尔零值为 `false`，`Load` 必须先构造带默认值的结构体再执行 `json.Unmarshal`；解析失败时须返回全新的默认值结构体，不可复用已被部分写入的对象。

`Save` 写入临时文件后 `rename`，避免中途崩溃留下截断的 JSON。

**位置**：`state.json` 存于 exe 同目录，路径通过 `os.Executable()` 获取。现有代码用 `os.Args[0]` 推断 exe 目录，从 PATH 启动时该值可能只是文件名，一并改用 `os.Executable()`。`state.json` 加入 `.gitignore`。

**兼容性**：既有 `config.toml` 中的 `[ui]` 段将被 go-toml v2 作为未知字段静默忽略，不报错、不影响加载。`show_log_on_startup` 会重置为默认值一次，点击托盘菜单即可恢复，不编写迁移代码。`config.toml.example` 无需修改（其中本就没有 `[ui]` 段）。

### 3. 托盘菜单

**`Tray` 结构体新增两个字段**，持有需要翻动勾选状态的两个 Action：

```go
actShowLog *walk.Action  // 显示日志
actStartup *walk.Action  // 日志启动时显示
```

**`NewTray` 改为接收回调集合**，不再依赖配置与状态：

```go
type Handlers struct {
    OnToggleLog     func()
    OnToggleStartup func()
    OnExit          func()
}

func NewTray(app *walk.Application, h Handlers) (*Tray, error)
```

`NewTray` 内部一次性构建三个 Action、绑定对应回调、加入 ContextMenu，此后不再修改菜单结构。"退出"不设置 `SetCheckable(true)`。

这一改动让 `tray` 包不再 import `config` 包——它退化为纯 UI 组件，只负责"显示菜单"和"报告菜单被点击"，如何响应由 main.go 编排。

**删除** `Tray.SetMenu` 方法和 `MenuItem` 类型——它们是"将固定菜单当作动态数据渲染"这一错误建模的产物。

**删除** 包级函数 `UpdateMenu`、`ToggleLogWindow`、`ToggleStartupSetting`、`CleanupAndExit`。这些编排逻辑移入 main.go，作为 `Handlers` 的实现。

**新增** `(*Tray).SyncMenuState(showLog, startup bool)`，仅执行两次 `SetChecked`。

**调用点变更清单：**

| 位置 | 现状 | 变更后 |
|---|---|---|
| main.go 初始化 | `tray.UpdateMenu(...)` | 删除——`NewTray` 已设初始状态 |
| main.go 双击回调 | `tray.ToggleLogWindow(...)` | 改为调用与 `OnToggleLog` 相同的本地函数 |
| main.go 热重载 | `tray.UpdateMenu(...)` | 删除——见下 |
| tray.go 两个 Toggle 函数末尾 | `UpdateMenu(...)` | 随函数一并移入 main.go，改调 `SyncMenuState` |

热重载路径的菜单同步调用删除：菜单状态只取决于日志窗口可见性与 `state.json`，与 `config.toml` 重载无关。

### 4. 文档更新

README 需要更新三处：

- `[ui] show_log_on_startup` 的说明改为"由托盘菜单控制，存于 `state.json`，默认启动时显示"
- 移除"需自行准备 `icon.ico` 放在 exe 同目录"的说明，图标已内嵌
- 新增图标重新生成流程：修改 `tools/genicon/` 后依次执行 `go run ./tools/genicon` 与上述 `rsrc` 命令，并提交产物

## 测试策略

**`state` 包——自动化单元测试：**

- 文件不存在时 `Load` 返回 `ShowLogOnStartup == true`
- JSON 内容损坏时 `Load` 返回默认值且不 panic
- 文件中显式写入 `false` 时 `Load` 正确返回 `false`
- `Save` 后 `Load` 往返一致
- `Save` 完成后目录中不残留 `.tmp` 文件

**`tray` 包——手动验证清单：**

walk 的 Action 与 NotifyIcon 依赖 Windows GUI 消息循环，无法在测试中构造，因此以手动清单覆盖：

1. 启动后托盘区显示绿色分叉图标，日志中无 walk 错误堆栈
2. 反复点击菜单项 20 次，菜单始终为三项
3. "退出"菜单项不带勾选框
4. 翻动"日志启动时显示"后重启程序，状态保持
5. 上述操作全程 `config.toml` 内容不变——操作前后比对文件哈希（`certutil -hashfile config.toml SHA256`），而非依赖 `git diff`（该文件当前已处于被重写状态）

**回归验证：** 现有 `config`、`handler`、`matcher`、`signal` 包的测试须全部通过。`config` 包中针对 `SaveConfig` 的测试随函数删除一并移除。

## 验收标准

1. 托盘区显示图标，启动日志中无图标加载错误
2. 菜单项数量恒为三，与点击次数无关
3. 任何操作都不修改 `config.toml`，文件的注释与格式保持原样
4. `show_log_on_startup` 的翻动结果在重启后保持
5. `state` 包单元测试与现有全部测试通过

## 范围外事项

以下问题在实现前需要处理或决策，但不属于本设计：

1. **工作区未提交改动**：`tray/logwindow.go` 存在半完成的修改（移除了窗口居中逻辑），`config.toml` 处于被程序重写后的状态。实现开始前须先提交或丢弃。
2. **config.toml 已丢失的内容**：注释与被注释掉的配置项已经丢失，本次修复不会恢复它们，需参照 `git show HEAD:config.toml` 手工补回。
3. **config.toml 包含真实 API key 且已被 git 跟踪**：`.gitignore` 中的 `config.toml` 一行对已跟踪文件无效，密钥已进入 3 个提交的历史。若仓库将来公开或推送至远端，这些密钥等同于已泄露。处理方式（`git rm --cached` 配合密钥轮换，或重写历史）需另行决策。
