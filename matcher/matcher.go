// matcher/matcher.go
package matcher

import (
	"errors"

	"github.com/gjj2828/TinyProxyAPI/config"
)

// ErrModelNotConfigured 模型不在 models 表中，且未配置 default platform
var ErrModelNotConfigured = errors.New("model not configured")

// ErrPlatformNotConfigured 配置的 default platform 在 platforms 表中找不到
var ErrPlatformNotFound = errors.New("default platform not found in platforms")

type Matcher struct {
	cfg *config.Config
}

func NewMatcher(cfg *config.Config) *Matcher {
	return &Matcher{cfg: cfg}
}

// Match 根据协议和模型名查找平台配置。
// 查找优先级：[models.<protocol>] 专用映射 > [models] 通用映射 > default platform。
// 若既不匹配且未配置 default platform，返回 ErrModelNotConfigured
// 若选定的 platform 在 platforms 表中不存在，返回 ErrPlatformNotFound
func (m *Matcher) Match(protocol, model string) (*config.Platform, error) {
	platformID, ok := m.lookupPlatformID(protocol, model)
	if !ok {
		if m.cfg.Default.Platform == "" {
			return nil, ErrModelNotConfigured
		}
		platformID = m.cfg.Default.Platform
	}
	platform, ok := m.cfg.Platforms[platformID]
	if !ok {
		return nil, ErrPlatformNotFound
	}
	return &platform, nil
}

// lookupPlatformID 依次在协议专用映射和通用映射中查找模型对应的平台 ID。
func (m *Matcher) lookupPlatformID(protocol, model string) (string, bool) {
	if byProtocol, ok := m.cfg.ModelsByProtocol[protocol]; ok {
		if platformID, ok := byProtocol[model]; ok {
			return platformID, true
		}
	}
	platformID, ok := m.cfg.Models[model]
	return platformID, ok
}

// GetAllowedKeys 返回全局 API Key 白名单
func (m *Matcher) GetAllowedKeys() []string {
	return m.cfg.AllowedKeys
}

// GetDefaultPlatform 返回配置的 default platform 名称
func (m *Matcher) GetDefaultPlatform() string {
	return m.cfg.Default.Platform
}

// GetModels 返回所有配置的模型映射 (modelID -> platformID)。
// 合并 [models] 通用映射与所有 [models.<protocol>] 协议覆盖；
// 同一模型同时出现在两处时以通用映射为准，避免 /v1/models 出现重复项。
func (m *Matcher) GetModels() map[string]string {
	merged := make(map[string]string, len(m.cfg.Models))
	for model, platformID := range m.cfg.Models {
		merged[model] = platformID
	}
	for _, byProtocol := range m.cfg.ModelsByProtocol {
		for model, platformID := range byProtocol {
			if _, exists := merged[model]; !exists {
				merged[model] = platformID
			}
		}
	}
	return merged
}
