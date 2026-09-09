// config/loader_test.go
package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	content := `
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
}