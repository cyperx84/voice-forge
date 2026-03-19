package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFileAtomic(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcPath := filepath.Join(srcDir, "test.txt")
	dstPath := filepath.Join(dstDir, "test.txt")

	content := []byte("hello world content for atomic copy test")
	if err := os.WriteFile(srcPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	got, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("reading copied file: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("copied content = %q, want %q", got, content)
	}
}

func TestCopyFileAtomicNoTempLeftover(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcPath := filepath.Join(srcDir, "test.ogg")
	dstPath := filepath.Join(dstDir, "test.ogg")

	if err := os.WriteFile(srcPath, []byte("fake audio"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := copyFile(srcPath, dstPath); err != nil {
		t.Fatal(err)
	}

	// Verify no temp files left behind
	entries, _ := os.ReadDir(dstDir)
	for _, e := range entries {
		if e.Name() != "test.ogg" {
			t.Errorf("leftover temp file found: %s", e.Name())
		}
	}
}

func TestCopyFileSrcNotExist(t *testing.T) {
	dstDir := t.TempDir()
	err := copyFile("/nonexistent/file", filepath.Join(dstDir, "out"))
	if err == nil {
		t.Error("expected error when source doesn't exist")
	}
}

func TestIsCorpusFile(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"recording.ogg", true},
		{"recording.wav", true},
		{"recording.mp3", true},
		{"recording.m4a", true},
		{"recording.txt", true},
		{"recording.pdf", false},
		{"recording.go", false},
		{"noext", false},
	}
	for _, tt := range tests {
		if got := isCorpusFile(tt.name); got != tt.want {
			t.Errorf("isCorpusFile(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}
