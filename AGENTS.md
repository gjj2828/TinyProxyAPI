# TinyProxyAPI 项目约定

## 概述

Windows 下的 LLM API 代理。Go + `lxn/walk` 系统托盘，转发 OpenAI / Anthropic 协议端点。
语言遵循全局 AGENTS.md：文档与注释用中文，commit message 用 Conventional Commits 英文格式
（`feat` / `fix` / `build` / `docs` / `chore`，已有历史即如此）。

## 构建：`icacls` 那一步不能删

`build.ps1` 结尾的 `icacls TinyProxyAPI.exe /setintegritylevel M` 是**功能性代码，不是可选项**。
删掉它，托盘图标在沙箱工作区内构建时必然失效。

- 原理：`IL = min(调用者 IL, 文件 IL)`，exe 继承工作区的 Low 标签 → Low IL 进程受 UIPI 限制
  → `Shell_NotifyIcon` 失败（`GetLastError() == 5`）。
- `/setintegritylevel` 只改 SACL 里的完整性 ACE，不动 DACL，因此不影响沙箱写入边界。
- **不要去改目录本身的标签**——那会破坏沙箱边界，且下次会话可能重新传播。
- 该步骤需要 Medium IL，必须在普通 PowerShell 里执行；在 DSH 工具通道里 `icacls` 会报
  `Access is denied`。脚本已做检测并给出警告，不要把这句警告当作噪音删掉。

## 两个不同的完整性级别问题，别混淆

它们对象不同、根因不同，`build.cmd` 的注释里专门标注了这点：

| | 对象 | 机制 | 表现 |
|---|---|---|---|
| 构建产物 | 跑起来的 **exe** | `IL = min(user, file)` | 托盘图标创建失败 |
| 构建脚本 | 被加载的 **ps1 文件** | PowerShell 判定 Low 标签文件为「非本地」→ `RemoteSigned` 要求签名 | `未对文件 build.ps1 进行数字签名` |

已实测确认：同一个 Low IL 进程，运行 Medium 标签的系统 `.ps1` 正常，
Low 标签的则被拒——**文件标签是唯一变量，与进程 IL 无关**。

因此 `build.cmd` 这个 wrapper 不可删：批处理文件不受 PowerShell 执行策略约束，
它用 `-ExecutionPolicy Bypass` 加载 `build.ps1`。顺带一提，`Unblock-File` 在这里无效
（只清 `Zone.Identifier`，而这些文件本来就没有该数据流）。

`%TEMP%` 下的文件同样带 Low 标签，不只是工作区。

## README 必须与构建脚本同步

`README.md` / `README_EN.md` 的「快速开始 → 运行」写的是构建入口。
改动 `build.ps1` 或 `build.cmd` 的调用方式时，**两个 README 都要同步更新**，
否则会给出用户跑不通的命令。中文改完记得改英文版。

## 端点相关

新增协议端点时注意：`allowed_keys` 必须是**顶层 key**，放在任何 `[table]` 之前。
写在 `[default]` 之后会被归入该表，而结构体没有对应字段 → 静默忽略 → 全部 key 被拒 → 401。
这个坑已经在两个 README 里标注，新增配置项时留意同类问题。
