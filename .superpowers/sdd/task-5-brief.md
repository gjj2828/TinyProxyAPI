# Task 5: SIGHUP 热重载

## 概述

实现 SIGHUP 信号监听，用于配置热重载。

## Global Constraints（必须遵守）

- Go 1.21+
- 监听地址：`:3178`
- 端点：`POST /v1/responses`
- 配置文件：`config.toml`（TOML 格式）
- 热重载：SIGHUP 信号
- 模型匹配：全量匹配

## 工作目录

`D:/Documents/Projects/TinyProxyAPI`

## 接口

```go
package signal

import (
    "os"
    "os/signal"
    "syscall"
)

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

## 具体任务

### Step 1: 编写测试

```go
// signal/sighup_test.go
package signal

import (
    "testing"
    "time"
)

func TestWatch(t *testing.T) {
    hup := Watch()

    // 发送 SIGHUP（需要手动测试，或使用 syscall.Syscall）
    // 这里只测试 channel 创建
    select {
    case <-hup:
        t.Fatal("unexpected signal")
    case <-time.After(100 * time.Millisecond):
        // OK
    }
}
```

### Step 2: 运行测试确认通过

Run: `go test ./signal/... -v`
Expected: PASS

### Step 3: 实现 Watch 函数

```go
// signal/sighup.go
package signal

import (
    "os"
    "os/signal"
    "syscall"
)

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

### Step 4: 运行测试确认通过

Run: `go test ./signal/... -v`
Expected: PASS

### Step 5: 提交

```bash
git add signal/sighup.go signal/sighup_test.go
git commit -m "feat: add SIGHUP watcher for hot reload"
```

## 验收标准

1. signal/sighup.go 存在，Watch 函数正确实现
2. signal/sighup_test.go 测试通过
3. 提交已创建
