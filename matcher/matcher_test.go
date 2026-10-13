// matcher/matcher_test.go
package matcher

import (
	"testing"

	"github.com/gjj2828/TinyProxyAPI/config"
)

func TestMatcher_Match(t *testing.T) {
	cfg := &config.Config{
		Default: config.DefaultConfig{
			Platform: "openai",
		},
		Platforms: map[string]config.Platform{
			"openai": {BaseURL: "https://api.openai.com", APIKey: "sk-test"},
		},
		Models: map[string]string{
			"gpt-4o": "openai",
		},
	}

	m := NewMatcher(cfg)

	// Test exact match
	p, err := m.Match("responses", "gpt-4o")
	if err != nil {
		t.Fatalf("expected match for gpt-4o, got error: %v", err)
	}
	if p.BaseURL != "https://api.openai.com" {
		t.Errorf("expected https://api.openai.com, got %s", p.BaseURL)
	}

	// Test fallback to default
	p, err = m.Match("responses", "unknown-model")
	if err != nil {
		t.Fatalf("expected fallback match, got error: %v", err)
	}
	if p.APIKey != "sk-test" {
		t.Errorf("expected sk-test, got %s", p.APIKey)
	}
}

func TestMatcher_MatchProtocolOverride(t *testing.T) {
	cfg := &config.Config{
		Platforms: map[string]config.Platform{
			"zn":        {BaseURL: "https://zn.example.com", APIKey: "sk-zn"},
			"anthropic": {BaseURL: "https://api.anthropic.com", APIKey: "sk-ant"},
		},
		// 通用映射：同名模型默认走 zn
		Models: map[string]string{
			"gpt-5": "zn",
		},
		// 协议覆盖：openai 协议下 gpt-5 改走 anthropic 平台
		ModelsByProtocol: map[string]map[string]string{
			"openai": {"gpt-5": "anthropic"},
		},
	}

	m := NewMatcher(cfg)

	// 未覆盖的协议 → 通用映射
	p, err := m.Match("responses", "gpt-5")
	if err != nil {
		t.Fatalf("expected match for responses/gpt-5, got error: %v", err)
	}
	if p.BaseURL != "https://zn.example.com" {
		t.Errorf("expected zn platform, got %s", p.BaseURL)
	}

	// 覆盖的协议 → 专用映射优先
	p, err = m.Match("openai", "gpt-5")
	if err != nil {
		t.Fatalf("expected match for openai/gpt-5, got error: %v", err)
	}
	if p.BaseURL != "https://api.anthropic.com" {
		t.Errorf("expected anthropic platform, got %s", p.BaseURL)
	}
}

func TestMatcher_GetAllowedKeys(t *testing.T) {
	cfg := &config.Config{
		Default: config.DefaultConfig{
			Platform: "openai",
		},
		AllowedKeys: []string{"key-001", "key-002", "key-003"},
		Platforms: map[string]config.Platform{
			"openai": {BaseURL: "https://api.openai.com", APIKey: "sk-test"},
		},
		Models: map[string]string{
			"gpt-4o": "openai",
		},
	}

	m := NewMatcher(cfg)

	keys := m.GetAllowedKeys()
	if len(keys) != 3 {
		t.Fatalf("expected 3 allowed keys, got %d", len(keys))
	}
	if keys[0] != "key-001" || keys[1] != "key-002" || keys[2] != "key-003" {
		t.Errorf("expected [key-001, key-002, key-003], got %v", keys)
	}
}
