## Task 6: 更新构建配置

**Files:**
- Modify: 构建脚本（如有）

- [ ] **Step 1: 确保构建命令使用 `-ldflags="-H windowsgui"`**

如果使用 go build，确保命令为：
```bash
go build -ldflags="-H windowsgui" -o TinyProxyAPI.exe
```

---

## 实现顺序

1. Task 1: 添加 walk 依赖
2. Task 2: 添加 UI 配置
3. Task 3: 创建托盘管理
4. Task 4: 创建日志窗口
5. Task 5: 修改 main.go
6. Task 6: 更新构建配置

---

## 注意事项

- walk 库需要在 Windows 环境编译
- 托盘图标需要准备 `icon.ico` 文件放在 exe 同目录，或内嵌资源
- 菜单项的勾选状态需要通过 `updateTrayMenu()` 实时刷新
- 配置文件保存使用 toml.Marshal，需要引入 toml 包
