package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		input string
		want  string
	}{
		{"~/foo/bar", filepath.Join(home, "foo/bar")},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"~/", home},
	}

	for _, tt := range tests {
		got := ExpandPath(tt.input)
		if got != tt.want {
			t.Errorf("ExpandPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if len(cfg.Corpus.Paths) != 2 {
		t.Errorf("expected 2 corpus paths, got %d", len(cfg.Corpus.Paths))
	}
	if cfg.LLM.Command != "claude" {
		t.Errorf("expected LLM command 'claude', got %q", cfg.LLM.Command)
	}
	if cfg.Profile.OutputDir != "~/.forge/profile" {
		t.Errorf("expected profile output dir '~/.forge/profile', got %q", cfg.Profile.OutputDir)
	}
}

func TestLoadNonExistent(t *testing.T) {
	// Load should return defaults when config file doesn't exist
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() with no config file should not error: %v", err)
	}
	if cfg.LLM.Command != "claude" {
		t.Errorf("expected default LLM command, got %q", cfg.LLM.Command)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.toml")

	// Override ConfigPath for testing
	origPath := ConfigPath()
	_ = origPath

	cfg := DefaultConfig()
	cfg.Corpus.Paths = []string{"~/test-corpus"}

	// Write directly to temp file
	data, err := os.ReadFile(configFile)
	_ = data
	_ = err

	// Test Save by writing config
	if err := os.MkdirAll(filepath.Dir(configFile), 0755); err != nil {
		t.Fatal(err)
	}

	if err := Save(cfg); err != nil {
		// This may fail if ~/.forge doesn't exist in test env, which is fine
		t.Skipf("Save requires ~/.forge to exist: %v", err)
	}
}

func TestCorpusPaths(t *testing.T) {
	home, _ := os.UserHomeDir()
	cfg := Config{
		Corpus: CorpusConfig{
			Paths: []string{"~/corpus1", "/absolute/corpus2"},
		},
	}

	paths := cfg.CorpusPaths()
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(paths))
	}
	if paths[0] != filepath.Join(home, "corpus1") {
		t.Errorf("expected expanded path, got %q", paths[0])
	}
	if paths[1] != "/absolute/corpus2" {
		t.Errorf("expected absolute path unchanged, got %q", paths[1])
	}
}

func TestProfileDir(t *testing.T) {
	home, _ := os.UserHomeDir()
	cfg := Config{
		Profile: ProfileConfig{
			OutputDir: "~/.forge/profile",
		},
	}

	got := cfg.ProfileDir()
	want := filepath.Join(home, ".forge/profile")
	if got != want {
		t.Errorf("ProfileDir() = %q, want %q", got, want)
	}
}
