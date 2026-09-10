// config/loader.go
package config

import (
	"os"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Server      ServerConfig
	Default     DefaultConfig
	AllowedKeys []string  `toml:"allowed_keys"`
	Platforms  map[string]Platform
	Models     map[string]string
	UI         UIConfig
}

type UIConfig struct {
	ShowLogOnStartup bool `toml:"show_log_on_startup"`
}

type ServerConfig struct {
	Addr string
}

type DefaultConfig struct {
	Platform string
}

type Platform struct {
	BaseURL string `toml:"base_url"`
	APIKey  string `toml:"api_key"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}