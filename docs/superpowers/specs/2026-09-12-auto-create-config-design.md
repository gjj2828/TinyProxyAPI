# 配置文件自动生成与托盘引导设计

## 概述

当配置文件 `config.toml` 不存在时，程序自动基于内嵌的示例模板创建初始配置文件，启动代理服务、系统托盘与热重载监听，并弹出引导提示框且自动唤起系统编辑器打开该配置文件供用户配置；同时在托盘右键菜单中常驻「打开配置文件」选项，用户修改并保存后自动通过既有的热重载机制生效。

## 背景与动机

1. **首次运行体验不友好**：目前当 `config.toml` 缺失时，程序直接弹出底层 `os.ReadFile` 英文错误框并终止进程。初次使用的用户需要手动到处寻找模板或自行编写 TOML 文件。
2. **单文件绿色运行受阻**：在仅分发单 exe 时，缺少配套的 `config.toml.example` 也会导致用户无从参考。
3. **日常配置查看不便**：目前修改配置需要用户自行去文件管理器定位目录并打开 `config.toml`，托盘菜单缺乏快捷打开配置文件的入口。

通过“**内嵌模板自动生成 + 托盘常驻 + 引导弹窗与自动唤起编辑器 + 托盘快捷打开**”，可以提供极佳的开箱即用体验。

## 核心行为规范

### 1. 模板内嵌与自动生成
- **内嵌示例模板**：使用 Go `//go:embed config.toml.example` 将配置模板打包进可执行文件。
- **文件检查与创建**：在解析得到绝对路径 `configPath` 后，检测文件是否存在：
  - 若文件已存在：不执行任何创建动作，直接进入加载流程。
  - 若文件不存在：优先尝试读取目标同目录下的物理 `config.toml.example`，若无则使用内嵌模板，以权限 `0644` 写入 `configPath`，并标记 `createdNewConfig = true`。
  - 若因目录无写入权限导致创建失败：调用 `showError("配置文件创建失败", ...)` 明确告知用户并终止退出。

### 2. 编辑器可靠唤起机制
- 在 Windows 环境下打开配置文件：
  - 优先调用 Win32 原生 `win.ShellExecute(..., "open", configPath, ...)`，使用系统关联的编辑器（如 VS Code、Sublime Text、Notepad 等）打开。
  - **无关联程序兜底**：若当前系统未关联 `.toml` 文件（`ShellExecute` 返回错误码），自动降级使用 `notepad.exe <configPath>` 打开，确保 100% 能够顺利展现给用户，不弹系统缺失关联错误。
  - 若唤起失败，仅记录 warning 日志，不阻塞主程序运行。

### 3. 首次启动交互与引导
- 当 `createdNewConfig == true` 时：
  1. 系统托盘、HTTP 代理服务及文件热重载监听（fsnotify）均已在后台就绪。
  2. 自动调用编辑器唤起机制打开新生成的 `config.toml`。
  3. 弹出信息提示框（`walk.MsgBox`，信息图标）：
     - **标题**：提示
     - **内容**：
       ```text
       未检测到配置文件，已为您自动生成默认配置文件 config.toml。

       代理服务已在系统托盘运行，并已为您打开配置文件，请填写您的 API Key 与模型配置后保存，保存后将自动热重载生效。
       ```
  4. 用户点击“确定”后弹窗关闭，程序继续在托盘常驻。用户在编辑器中保存修改后，后台热重载监听器立即生效。

### 4. 托盘菜单增强
- 托盘右键菜单在顶部新增「打开配置文件」项，点击即可唤起系统编辑器打开当前正在使用的 `config.toml`。
- 菜单布局调整为：
  - **打开配置文件**
  - *(分隔线)*
  - **显示日志**（复选框）
  - **日志启动时显示**（复选框）
  - *(分隔线)*
  - **退出**

## 代码架构设计

### 1. `config` 包
- **`config/template.go`**：
  ```go
  package config

  import (
      _ "embed"
      "os"
      "path/filepath"
  )

  //go:embed config.toml.example
  var DefaultConfigTemplate []byte

  // EnsureConfigFile 检查目标路径配置文件是否存在；若不存在则自动从同目录 example 或内嵌模板生成。
  // 返回 created 标识是否为本次新创建。
  func EnsureConfigFile(path string) (bool, error) {
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
          content = DefaultConfigTemplate
      }

      if err := os.WriteFile(path, content, 0644); err != nil {
          return false, err
      }
      return true, nil
  }
  ```
- **注意文件移动/链接**：`config.toml.example` 位于项目根目录，在 Go 1.16+ `embed` 中，只能嵌入当前包目录或其子目录中的文件。因此可将 `config.toml.example` 移至 `config/config.toml.example`，或在根目录保留并通过拷贝/符号链接维护，或统一规范放置于 `config/config.toml.example`（并在根目录保留或同步）。

### 2. `tray` 包
- **`tray/tray.go`**：
  - `Handlers` 增加 `OnOpenConfig func()`。
  - 在菜单构建中添加 `actOpenConfig` 和 `walk.NewSeparatorAction()`。

### 3. `main.go`
- **`openConfigFile(path string)`**：
  封装 Windows 打开指定配置文件的辅助函数（`win.ShellExecute` + `notepad.exe` 降级）。
- **编排逻辑**：
  1. 解析并规范化 `configPath`。
  2. 调用 `config.EnsureConfigFile(configPath)` 得到 `created, err`。
  3. 若 `err != nil`，调用 `showError` 并退出。
  4. 正常调用 `config.LoadConfig(configPath)` 并启动托盘、HTTP 代理及热重载监听。
  5. 绑定托盘菜单的 `OnOpenConfig: func() { openConfigFile(configPath) }`。
  6. 若 `created == true`，调用 `openConfigFile(configPath)` 并显示引导 `walk.MsgBox`。
  7. 启动 `tr.MainWindow().Run()`。

### 4. 文档更新
- 更新 `README.md` 与 `README_EN.md`，增加首次运行自动生成配置与托盘菜单“打开配置文件”的说明。

## 边界与异常处理

1. **目录只读 / 权限不足**：若目标路径无写入权限导致无法创建文件，明确弹窗提示并退出，绝不静默失败。
2. **语法错误配置的保护**：若 `config.toml` 已经存在但格式错误，维持现有严格报错逻辑，绝不覆盖或抹除用户配置。
3. **安全拦截**：自动生成的模板中默认包含占位 `allowed_keys` 与非真实 API Key，在用户修改保存前外部请求会被 401 拒绝，不会误触发未配置上游请求。
4. **编辑器唤起降级**：`win.ShellExecute` 失败时自动降级到 `notepad.exe`；即使记事本也因极端异常无法启动，仅输出 warn 日志，服务正常保持运行。

## 测试验证策略

### 1. 自动化单元测试
- **`config/template_test.go`**：
  - 测试 `DefaultConfigTemplate` 非空，且可通过 `toml.Unmarshal` 正确解析为有效的 `Config` 结构体。
  - 测试 `EnsureConfigFile`：
    - 在临时目录中测试不存在时成功创建文件，且内容与模板一致，返回 `created == true`。
    - 当文件已存在且包含自定义内容时，调用 `EnsureConfigFile` 返回 `created == false`，且已有内容未被覆盖改动。

### 2. 手动端到端验证
1. 临时重命名当前工作区的 `config.toml`。
2. 启动 `TinyProxyAPI.exe`。
3. 验证是否：
   - 自动生成了 `config.toml`；
   - 自动拉起编辑器展示 `config.toml`；
   - 弹出友好的中文引导提示框；
   - 托盘图标正常显示，HTTP 服务正常监听端口；
   - 点击右键菜单中的「打开配置文件」可正常唤起编辑器；
   - 在编辑器中修改配置后保存，日志窗口能看到 `reloading config... config reloaded successfully`；
   - 发送代理请求测试功能是否正常生效。
