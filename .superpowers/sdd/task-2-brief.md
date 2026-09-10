## Task 2: 添加 UI 配置

**Files:**
- Modify: `config/loader.go`
- Modify: `config.toml`

**Interfaces:**
- Consumes: toml 配置文件
- Produces: `Config.UI.ShowLogOnStartup` 字段（bool 类型，默认为 false）

- [ ] **Step 1: 在 Config 结构体中添加 UI 配置**

在 `config/loader.go` 中添加：

```go
type Config struct {
    Server      ServerConfig
    Default     DefaultConfig
    AllowedKeys []string  `toml:"allowed_keys"`
    Platforms  map[string]Platform
    Models     map[string]string
    UI         UIConfig
}

type UIConfig struct {
    ShowLogOnStartup bool `toml:"show_log_on_startup"`
}
```

- [ ] **Step 2: 在 config.toml 中添加 [ui] section**

```toml
[ui]
# 启动时是否显示日志窗口（默认 false）
show_log_on_startup = false
```

- [ ] **Step 3: 提交**

```bash
git add config/loader.go config.toml
git commit -m "config: add [ui] section with show_log_on_startup"
```

---

