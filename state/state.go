// state/state.go
// 持久化界面状态（独立于用户手工维护的 config.toml）。
// config.toml 含注释与格式，由用户维护，绝不能被程序回写；
// 这里只放与界面行为有关、丢失也不影响代理功能的设置。
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// State 是持久化的界面状态。
type State struct {
	ShowLogOnStartup bool `json:"show_log_on_startup"`
}

// 默认状态。日志窗口默认在启动时弹出。
func defaultState() *State {
	return &State{ShowLogOnStartup: true}
}

// Load 读取状态文件。
//
// 与 config.LoadConfig 不同：本函数不返回 error。文件缺失、JSON 损坏、
// 权限错误一律返回默认值并继续运行。配置缺失则无法工作，状态缺失只影响
// 日志窗口是否弹出，不值得让程序退出。
//
// 先构造默认值再 Unmarshal：Go 的 bool 零值是 false，若先声明空结构体再
// 解析，一个只写了其它字段的 JSON 会把 ShowLogOnStartup 错误地解析成 false。
// 解析失败时返回全新的默认结构，而不是已写入部分的残缺值。
func Load(path string) *State {
	s := defaultState()

	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}

	if err := json.Unmarshal(data, s); err != nil {
		return defaultState()
	}

	return s
}

// DefaultPath 返回状态文件的默认存储路径 ~/.tinyproxyapi/state.json。
// 若获取用户家目录失败，将优雅回退至可执行文件所在目录或系统临时目录。
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		return filepath.Join(home, ".tinyproxyapi", "state.json"), nil
	}

	// 降级策略：尝试可执行文件所在目录
	if exe, exeErr := os.Executable(); exeErr == nil {
		return filepath.Join(filepath.Dir(exe), "state.json"), nil
	}

	// 最终兜底：系统临时目录
	return filepath.Join(os.TempDir(), ".tinyproxyapi", "state.json"), nil
}

// Save 以原子方式写入状态文件。
//
// 先确保父目录存在，然后写同目录下的临时文件，再 Rename 覆盖目标，
// 保证写入中途崩溃不会留下被截断的 JSON。成功后不留下任何临时文件。
func Save(path string, s *State) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.Marshal(s)
	if err != nil {
		return err
	}

	tmp := filepath.Join(dir, filepath.Base(path)+".tmp")
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}

	return nil
}
