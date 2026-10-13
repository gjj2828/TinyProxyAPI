// config/loader.go
package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	Server      ServerConfig
	Default     DefaultConfig
	AllowedKeys []string `toml:"allowed_keys"`
	Platforms   map[string]Platform
	// Models 是协议无关的默认映射：model -> platform。
	Models map[string]string
	// ModelsByProtocol 是按协议覆盖的映射：protocol -> model -> platform。
	// 通过 [models.<protocol>] 段落配置，优先级高于 Models。
	ModelsByProtocol map[string]map[string]string
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

// rawConfig 用于解析 config.toml。
// [models] 段是混合类型：既可能是扁平的 "model" = "platform"，
// 也可能是按协议分组的 [models.<protocol>] 子表，因此用 map[string]any 接收后归一化。
type rawConfig struct {
	Server      ServerConfig
	Default     DefaultConfig
	AllowedKeys []string `toml:"allowed_keys"`
	Platforms   map[string]Platform
	Models      map[string]any
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var raw rawConfig
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	cfg := &Config{
		Server:           raw.Server,
		Default:          raw.Default,
		AllowedKeys:      raw.AllowedKeys,
		Platforms:        raw.Platforms,
		Models:           make(map[string]string),
		ModelsByProtocol: make(map[string]map[string]string),
	}

	for key, val := range raw.Models {
		switch v := val.(type) {
		case string:
			// 扁平写法：[models] 下的 "model" = "platform"
			cfg.Models[key] = v
		case map[string]any:
			// 协议覆盖写法：[models.<protocol>] 下的 "model" = "platform"
			inner := make(map[string]string, len(v))
			for model, platform := range v {
				s, ok := platform.(string)
				if !ok {
					return nil, fmt.Errorf("config: [models.%s].%s must be a platform id string", key, model)
				}
				inner[model] = s
			}
			cfg.ModelsByProtocol[key] = inner
		default:
			return nil, fmt.Errorf("config: invalid [models] entry %q", key)
		}
	}

	return cfg, nil
}
