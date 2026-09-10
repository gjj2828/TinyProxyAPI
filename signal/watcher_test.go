package signal

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWatchConfig_FiresOnFileChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("initial"), 0o644); err != nil {
		t.Fatal(err)
	}

	ch, err := WatchConfig(path)
	if err != nil {
		t.Fatalf("WatchConfig failed: %v", err)
	}

	// 修改文件应该触发事件（带 100ms 防抖）
	if err := os.WriteFile(path, []byte("updated"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case <-ch:
		// 期望事件到达
	case <-time.After(2 * time.Second):
		t.Fatal("expected reload signal within 2s after file change")
	}
}

func TestWatchConfig_Debounces(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("initial"), 0o644); err != nil {
		t.Fatal(err)
	}

	ch, err := WatchConfig(path)
	if err != nil {
		t.Fatalf("WatchConfig failed: %v", err)
	}

	// 快速连续多次写入 → 防抖应该只产生一次触发
	for range 5 {
		if err := os.WriteFile(path, []byte("updated"), 0o644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// 等待防抖窗口结束
	time.Sleep(300 * time.Millisecond)

	// 此时 channel 应该是空的（已经被 select-default 跳过），且不会再有新事件
	select {
	case <-ch:
		// 触发了一次，这没问题
	default:
		// 也没问题
	}

	// 关键断言：200ms 内不应该再有事件
	select {
	case <-ch:
		t.Fatal("unexpected extra event after debounce window")
	case <-time.After(200 * time.Millisecond):
		// 期望没有事件
	}
}

func TestWatchConfig_IgnoresOtherFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	other := filepath.Join(dir, "other.toml")
	if err := os.WriteFile(path, []byte("initial"), 0o644); err != nil {
		t.Fatal(err)
	}

	ch, err := WatchConfig(path)
	if err != nil {
		t.Fatalf("WatchConfig failed: %v", err)
	}

	// 修改其他文件不应触发
	if err := os.WriteFile(other, []byte("updated"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case <-ch:
		t.Fatal("watcher fired for unrelated file")
	case <-time.After(300 * time.Millisecond):
		// 期望无事件
	}
}