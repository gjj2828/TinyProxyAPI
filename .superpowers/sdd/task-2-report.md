## Task 2 Report: 添加 UI 配置

**Status:** DONE

**Commits:**
- `22430bc` - config: add [ui] section with show_log_on_startup

**Changes:**
- `config/loader.go`: 添加了 `UIConfig` struct 和 `Config.UI` 字段
- `config.toml`: 添加了 `[ui]` section，包含 `show_log_on_startup = false`

**Test summary:** 无测试需要运行（仅为结构体添加）

**Concerns:** 无
