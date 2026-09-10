## Task 1: 添加 walk 依赖

**Files:**
- Modify: `go.mod`

- [ ] **Step 1: 添加 walk 依赖**

Run: `go get github.com/lxn/walk@latest`
Expected: go.mod 添加 `github.com/lxn/walk v0.8.0` 和 `github.com/lxn/win v0.21.0`

- [ ] **Step 2: 提交**

```bash
git add go.mod go.sum
git commit -m "deps: add walk for system tray support"
```

---

