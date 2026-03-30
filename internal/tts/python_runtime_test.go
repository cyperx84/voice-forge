package tts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePythonRuntimeUsesEnvVar(t *testing.T) {
	t.Setenv("FORGE_TEST_RUNTIME", "/tmp/custom-python")
	r := ResolveConfiguredRuntime("FORGE_TEST_RUNTIME", "", ".forge/test")
	if r.PythonPath != "/tmp/custom-python/bin/python3" {
		t.Fatalf("got %q", r.PythonPath)
	}
}

func TestResolvePythonRuntimeUsesConfigPath(t *testing.T) {
	r := ResolveConfiguredRuntime("FORGE_TEST_RUNTIME_MISSING", "/opt/f5-venv", ".forge/test")
	if r.PythonPath != "/opt/f5-venv/bin/python3" {
		t.Fatalf("got %q", r.PythonPath)
	}
}

func TestResolvePythonRuntimeDefaultDir(t *testing.T) {
	home := t.TempDir()
	oldHome := os.Getenv("HOME")
	t.Cleanup(func() { _ = os.Setenv("HOME", oldHome) })
	_ = os.Setenv("HOME", home)
	r := ResolveConfiguredRuntime("FORGE_TEST_RUNTIME_MISSING", "", ".forge/chatterbox")
	want := filepath.Join(home, ".forge", "chatterbox", "bin", "python3")
	if r.PythonPath != want {
		t.Fatalf("got %q want %q", r.PythonPath, want)
	}
}
