// matcher/matcher_test.go
package matcher

import (
	"testing"

	"github.com/user/TinyProxyAPI/config"
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
	p, ok := m.Match("gpt-4o")
	if !ok {
		t.Fatal("expected match for gpt-4o")
	}
	if p.BaseURL != "https://api.openai.com" {
		t.Errorf("expected https://api.openai.com, got %s", p.BaseURL)
	}

	// Test fallback to default
	p, ok = m.Match("unknown-model")
	if !ok {
		t.Fatal("expected fallback match")
	}
	if p.APIKey != "sk-test" {
		t.Errorf("expected sk-test, got %s", p.APIKey)
	}
}
