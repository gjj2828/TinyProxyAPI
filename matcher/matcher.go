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

// Match 根据模型名查找平台配置
// 优先全量匹配 models 表，未找到则使用 default platform
// 若未配置 default platform 且模型不在 models 表中，返回 ErrModelNotConfigured
// 若配置了 default platform 但 platforms 表中没有该配置，返回 ErrPlatformNotFound
func (m *Matcher) Match(model string) (*config.Platform, error) {
	platformID, ok := m.cfg.Models[model]
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

// GetAllowedKeys 返回全局 API Key 白名单
func (m *Matcher) GetAllowedKeys() []string {
	return m.cfg.AllowedKeys
}

// GetDefaultPlatform 返回配置的 default platform 名称
func (m *Matcher) GetDefaultPlatform() string {
	return m.cfg.Default.Platform
}

// GetModels 返回所有配置的模型映射 (modelID -> platformID)
func (m *Matcher) GetModels() map[string]string {
	return m.cfg.Models
}
