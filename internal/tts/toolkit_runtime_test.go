package tts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestToolkitCheckMissingPath(t *testing.T) {
	tk := &ToolkitBackend{Path: "/definitely/missing"}
	if err := tk.Check(); err == nil {
		t.Fatal("expected missing path error")
	}
}

func TestToolkitCheckMissingExecutable(t *testing.T) {
	dir := t.TempDir()
	tk := &ToolkitBackend{Path: dir}
	if err := tk.Check(); err == nil {
		t.Fatal("expected missing executable error")
	}
}

func TestToolkitCheckHappyPath(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "tts-toolkit")
	content := "#!/bin/sh\necho 'tts-toolkit help'\n"
	if err := os.WriteFile(script, []byte(content), 0755); err != nil {
		t.Fatal(err)
	}
	tk := &ToolkitBackend{Path: dir}
	if err := tk.Check(); err != nil {
		t.Fatalf("expected toolkit check to pass, got %v", err)
	}
}

func TestKokoroSetupIncludesToolkitFailure(t *testing.T) {
	k := &KokoroBackend{Toolkit: &ToolkitBackend{Path: "/missing"}}
	if err := k.Setup(); err == nil {
		t.Fatal("expected setup error")
	}
}
