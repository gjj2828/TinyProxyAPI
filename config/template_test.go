// config/template_test.go
package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureConfigFile_NotExists(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	fallback := []byte("fallback = true\n")

	created, err := EnsureConfigFile(configPath, fallback)
	if err != nil {
		t.Fatalf("EnsureConfigFile failed: %v", err)
	}
	if !created {
		t.Errorf("expected created to be true, got false")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read created config file: %v", err)
	}
	if !bytes.Equal(data, fallback) {
		t.Errorf("expected content %q, got %q", string(fallback), string(data))
	}
}

func TestEnsureConfigFile_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	existingContent := []byte("existing = true\n")
	if err := os.WriteFile(configPath, existingContent, 0644); err != nil {
		t.Fatalf("failed to write existing config file: %v", err)
	}

	fallback := []byte("fallback = true\n")
	created, err := EnsureConfigFile(configPath, fallback)
	if err != nil {
		t.Fatalf("EnsureConfigFile failed: %v", err)
	}
	if created {
		t.Errorf("expected created to be false, got true")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}
	if !bytes.Equal(data, existingContent) {
		t.Errorf("expected content %q, got %q", string(existingContent), string(data))
	}
}

func TestEnsureConfigFile_PrefersDiskExample(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	examplePath := filepath.Join(dir, "config.toml.example")
	exampleContent := []byte("example = true\n")
	fallback := []byte("fallback = true\n")

	if err := os.WriteFile(examplePath, exampleContent, 0644); err != nil {
		t.Fatalf("failed to write example config file: %v", err)
	}

	created, err := EnsureConfigFile(configPath, fallback)
	if err != nil {
		t.Fatalf("EnsureConfigFile failed: %v", err)
	}
	if !created {
		t.Errorf("expected created to be true, got false")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}
	if !bytes.Equal(data, exampleContent) {
		t.Errorf("expected content %q, got %q", string(exampleContent), string(data))
	}
}

func TestEnsureConfigFile_EmptyDiskExampleFallsBack(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.toml")
	examplePath := filepath.Join(dir, "config.toml.example")
	fallback := []byte("fallback = true\n")

	if err := os.WriteFile(examplePath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write empty example config file: %v", err)
	}

	created, err := EnsureConfigFile(configPath, fallback)
	if err != nil {
		t.Fatalf("EnsureConfigFile failed: %v", err)
	}
	if !created {
		t.Errorf("expected created to be true, got false")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read config file: %v", err)
	}
	if !bytes.Equal(data, fallback) {
		t.Errorf("expected content %q, got %q", string(fallback), string(data))
	}
}

func TestEnsureConfigFile_WriteError(t *testing.T) {
	dir := t.TempDir()
	badPath := filepath.Join(dir, "nonexistent", "sub", "config.toml")
	created, err := EnsureConfigFile(badPath, []byte("test"))
	if err == nil {
		t.Errorf("expected error, got nil")
	}
	if created {
		t.Errorf("expected created to be false on error")
	}
}

