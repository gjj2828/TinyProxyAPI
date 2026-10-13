// config/loader_test.go
package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	content := `
allowed_keys = ["key-001", "key-002", "key-003"]

[server]
addr = ":3178"

[default]
platform = "openai"

[platforms]
"openai" = { base_url = "https://api.openai.com", api_key = "sk-test" }

[models]
"gpt-4o" = "openai"
`
	tmpfile, err := os.CreateTemp("", "config-*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Server.Addr != ":3178" {
		t.Errorf("expected addr :3178, got %s", cfg.Server.Addr)
	}
	if cfg.Default.Platform != "openai" {
		t.Errorf("expected default platform openai, got %s", cfg.Default.Platform)
	}
	if cfg.Platforms["openai"].BaseURL != "https://api.openai.com" {
		t.Errorf("expected openai base_url https://api.openai.com, got %s", cfg.Platforms["openai"].BaseURL)
	}
	if cfg.Models["gpt-4o"] != "openai" {
		t.Errorf("expected gpt-4o -> openai, got %s", cfg.Models["gpt-4o"])
	}
	if len(cfg.AllowedKeys) != 3 {
		t.Errorf("expected 3 allowed keys, got %d", len(cfg.AllowedKeys))
	}
	if cfg.AllowedKeys[0] != "key-001" {
		t.Errorf("expected first allowed key key-001, got %s", cfg.AllowedKeys[0])
	}
}

func TestLoadConfig_ModelsByProtocol(t *testing.T) {
	content := `
[platforms]
"zn" = { base_url = "https://zn.example.com", api_key = "sk-zn" }
"anthropic" = { base_url = "https://api.anthropic.com", api_key = "sk-ant" }

# 通用映射（向后兼容的扁平写法）
[models]
"gpt-5" = "zn"
"gpt-4o" = "zn"

# 协议级覆盖
[models.openai]
"gpt-5" = "anthropic"

[models.anthropic]
"claude-sonnet-4" = "anthropic"
`
	tmpfile, err := os.CreateTemp("", "config-*.toml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpfile.Close()

	cfg, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// 扁平写法进入 Models
	if cfg.Models["gpt-5"] != "zn" {
		t.Errorf("expected models.gpt-5 -> zn, got %s", cfg.Models["gpt-5"])
	}
	if cfg.Models["gpt-4o"] != "zn" {
		t.Errorf("expected models.gpt-4o -> zn, got %s", cfg.Models["gpt-4o"])
	}
	// 协议覆盖进入 ModelsByProtocol
	if got := cfg.ModelsByProtocol["openai"]["gpt-5"]; got != "anthropic" {
		t.Errorf("expected models.openai.gpt-5 -> anthropic, got %s", got)
	}
	if got := cfg.ModelsByProtocol["anthropic"]["claude-sonnet-4"]; got != "anthropic" {
		t.Errorf("expected models.anthropic.claude-sonnet-4 -> anthropic, got %s", got)
	}
	// 协议覆盖不应污染通用映射
	if _, ok := cfg.Models["claude-sonnet-4"]; ok {
		t.Errorf("did not expect claude-sonnet-4 in flat Models")
	}
}
