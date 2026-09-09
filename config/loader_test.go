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