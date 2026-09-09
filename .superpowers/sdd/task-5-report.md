# Task 5: SIGHUP 热重载 - 报告

## 完成状态

**Status:** DONE

## 实现内容

### 1. 创建的文件

- `D:/Documents/Projects/TinyProxyAPI/signal/sighup.go` - Watch 函数实现
- `D:/Documents/Projects/TinyProxyAPI/signal/sighup_test.go` - 单元测试

### 2. Watch 函数实现

```go
func Watch() chan struct{} {
    hup := make(chan struct{}, 1)
    sigs := make(chan os.Signal, 1)
    signal.Notify(sigs, syscall.SIGHUP)

    go func() {
        <-sigs
        hup <- struct{}{}
    }()

    return hup
}
```

功能：
- 创建带缓冲的 `chan struct{}` 用于通知
- 监听 `syscall.SIGHUP` 信号
- goroutine 等待信号并通过 channel 通知调用方

### 3. 测试结果

```
=== RUN   TestWatch
--- PASS: TestWatch (0.10s)
PASS
```

测试通过，验证了 channel 创建正常。

## 提交信息

- **Commit:** 48f7825
- **Subject:** feat: add SIGHUP watcher for hot reload

## 验收标准检查

- [x] signal/sighup.go 存在，Watch 函数正确实现
- [x] signal/sighup_test.go 测试通过
- [x] 提交已创建
