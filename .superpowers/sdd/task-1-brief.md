### Task 1: ConfigLoader 支持 AllowedKeys

**Files:**
- Modify: `config/loader.go:10-15`
- Test: `config/loader_test.go`

**Interfaces:**
- Produces: `Config.AllowedKeys []string`

- [ ] **Step 1: 修改 Config 结构体，添加 AllowedKeys 字段**

```go
type Config struct {
    Server     ServerConfig
    Default    DefaultConfig
    AllowedKeys []string      // 新增
    Platforms  map[string]Platform
    Models     map[string]string
}
```

- [ ] **Step 2: 运行现有测试，确认不影响现有功能**

Run: `go test ./config/... -v`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add config/loader.go
git commit -m "feat: add AllowedKeys field to Config"
```

---

