### Task 2: Matcher 添加 GetAllowedKeys 方法

**Files:**
- Modify: `matcher/matcher.go`
- Test: `matcher/matcher_test.go`

**Interfaces:**
- Consumes: `*config.Config`
- Produces: `GetAllowedKeys() []string`

- [ ] **Step 1: 在 matcher.go 末尾添加 GetAllowedKeys 方法**

```go
// GetAllowedKeys 返回全局 API Key 白名单
func (m *Matcher) GetAllowedKeys() []string {
    return m.cfg.AllowedKeys
}
```

- [ ] **Step 2: 运行现有测试，确认不影响现有功能**

Run: `go test ./matcher/... -v`
Expected: PASS

- [ ] **Step 3: 提交**

```bash
git add matcher/matcher.go
git commit -m "feat: add GetAllowedKeys method to Matcher"
```

---

