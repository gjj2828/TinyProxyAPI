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