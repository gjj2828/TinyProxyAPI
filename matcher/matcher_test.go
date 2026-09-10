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
	p, err := m.Match("gpt-4o")
	if err != nil {
		t.Fatalf("expected match for gpt-4o, got error: %v", err)
	}
	if p.BaseURL != "https://api.openai.com" {
		t.Errorf("expected https://api.openai.com, got %s", p.BaseURL)
	}

	// Test fallback to default
	p, err = m.Match("unknown-model")
	if err != nil {
		t.Fatalf("expected fallback match, got error: %v", err)
	}
	if p.APIKey != "sk-test" {
		t.Errorf("expected sk-test, got %s", p.APIKey)
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
