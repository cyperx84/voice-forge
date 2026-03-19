package config

import (
	"os"
	"path/filepath"
	"testing"

	toml "github.com/pelletier/go-toml/v2"
)

func TestSaveFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.toml")

	cfg := DefaultConfig()
	data, err := toml.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(configFile, data, 0600); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(configFile)
	if err != nil {
		t.Fatal(err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("config file permissions = %o, want 0600", perm)
	}
}
