// matcher/matcher.go
package matcher

import "github.com/user/TinyProxyAPI/config"

type Matcher struct {
	cfg *config.Config
}

func NewMatcher(cfg *config.Config) *Matcher {
	return &Matcher{cfg: cfg}
}

// Match 根据模型名查找平台配置
// 优先全量匹配 models 表，未找到则使用 default platform
func (m *Matcher) Match(model string) (*config.Platform, bool) {
	platformID, ok := m.cfg.Models[model]
	if !ok {
		platformID = m.cfg.Default.Platform
	}
	platform, ok := m.cfg.Platforms[platformID]
	return &platform, ok
}
