package signal

import (
	"os"
	"os/signal"
	"syscall"
)

// Watch 注册 SIGHUP 信号监听并返回 channel。
// 每次收到 SIGHUP 时会向 channel 发送一个空结构体。
// 注意：在 Windows 上 SIGHUP 不会被触发（Go runtime 不支持），
// 此时应配合 WatchConfig 使用文件 watcher 实现热重载。
func Watch() chan struct{} {
	hup := make(chan struct{}, 1)
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGHUP)

	go func() {
		for range sigs {
			select {
			case hup <- struct{}{}:
			default:
			}
		}
	}()

	return hup
}