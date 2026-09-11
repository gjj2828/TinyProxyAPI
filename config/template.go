// config/template.go
package config

import (
	"os"
	"path/filepath"
)

// EnsureConfigFile checks if the configuration file at path exists.
// If not, it populates it using a disk example (config.toml.example in the same directory)
// if present and non-empty, or fallbackTemplate otherwise.
// Returns (true, nil) if a new config file was created, or (false, nil) if it already existed.
func EnsureConfigFile(path string, fallbackTemplate []byte) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return false, nil
	}
	if !os.IsNotExist(err) {
		return false, err
	}

	content := fallbackTemplate
	examplePath := filepath.Join(filepath.Dir(path), "config.toml.example")
	if exampleData, readErr := os.ReadFile(examplePath); readErr == nil && len(exampleData) > 0 {
		content = exampleData
	}

	if err := os.WriteFile(path, content, 0644); err != nil {
		return false, err
	}

	return true, nil
}
