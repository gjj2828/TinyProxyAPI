// state/state_test.go
package state

import (
	"os"
	"path/filepath"
	"testing"
)

// 文件不存在时应返回默认值，且不报错。
func TestLoadDefaultWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	s := Load(path)

	if s == nil {
		t.Fatal("Load 在文件缺失时返回了 nil")
	}
	if !s.ShowLogOnStartup {
		t.Errorf("ShowLogOnStartup 默认值应为 true，实际为 %v", s.ShowLogOnStartup)
	}
}

// JSON 损坏时应返回全新默认值，而不是半写入的残缺结构。
func TestLoadDefaultWhenInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	if err := os.WriteFile(path, []byte("{not valid json at all"), 0644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}

	s := Load(path)

	if s == nil {
		t.Fatal("Load 在 JSON 损坏时返回了 nil")
	}
	if !s.ShowLogOnStartup {
		t.Errorf("ShowLogOnStartup 应为全新默认值 true，实际为 %v", s.ShowLogOnStartup)
	}
}

// 文件中显式写入 false 时，应读出 false 而不是被默认值覆盖。
func TestLoadFalseValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	s := &State{ShowLogOnStartup: false}
	if err := Save(path, s); err != nil {
		t.Fatalf("Save 失败: %v", err)
	}

	loaded := Load(path)
	if loaded.ShowLogOnStartup {
		t.Errorf("ShowLogOnStartup 应读出 false，实际为 %v", loaded.ShowLogOnStartup)
	}
}

// Save 后再 Load 应得到相同的值。
func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	for _, want := range []bool{true, false} {
		if err := Save(path, &State{ShowLogOnStartup: want}); err != nil {
			t.Fatalf("Save(%v) 失败: %v", want, err)
		}

		loaded := Load(path)
		if loaded.ShowLogOnStartup != want {
			t.Errorf("round trip: 期望 %v，实际 %v", want, loaded.ShowLogOnStartup)
		}
	}
}

// 文件存在但缺少键时（如空对象 {}），应读出默认值 true。
// 这验证"先构造默认值再 Unmarshal"的顺序：若用零值结构体直接 Unmarshal，
// 缺失的键不会触发任何赋值，ShowLogOnStartup 会错误地保持 false。
func TestLoadMissingKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatalf("写入测试文件失败: %v", err)
	}

	s := Load(path)

	if s == nil {
		t.Fatal("Load 返回了 nil")
	}
	if !s.ShowLogOnStartup {
		t.Errorf("缺失 show_log_on_startup 键时应读出默认值 true，实际为 %v", s.ShowLogOnStartup)
	}
}

// Save 成功后不应在同目录留下任何 .tmp 残留文件。
func TestSaveLeavesNoTempFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	if err := Save(path, &State{ShowLogOnStartup: true}); err != nil {
		t.Fatalf("Save 失败: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读取目录失败: %v", err)
	}

	for _, entry := range entries {
		if entry.Type().IsRegular() && filepath.Ext(entry.Name()) == ".tmp" {
			t.Errorf("Save 留下了临时文件: %s", entry.Name())
		}
	}
}
