package ingest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cyperx84/voice-forge/internal/corpus"
)

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		ext  string
		want string
	}{
		{".go", "go"},
		{".py", "python"},
		{".ts", "typescript"},
		{".tsx", "typescript"},
		{".rs", "rust"},
		{".java", "java"},
		{".md", ""},
		{".txt", ""},
	}
	for _, tt := range tests {
		got := DetectLanguage(tt.ext)
		if got != tt.want {
			t.Errorf("DetectLanguage(%q) = %q, want %q", tt.ext, got, tt.want)
		}
	}
}

func TestIsCodeFile(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"main.go", true},
		{"app.py", true},
		{"index.ts", true},
		{"readme.md", false},
		{"data.json", false},
	}
	for _, tt := range tests {
		got := IsCodeFile(tt.name)
		if got != tt.want {
			t.Errorf("IsCodeFile(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIngestCodeFile(t *testing.T) {
	db := testDB(t)
	corpusRoot := t.TempDir()

	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "main.go")
	os.WriteFile(srcPath, []byte("package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n"), 0644)

	item, err := IngestCodeFile(db, corpusRoot, srcPath, CodeOptions{
		Source:   "test-project",
		Language: "go",
	})
	if err != nil {
		t.Fatalf("IngestCodeFile: %v", err)
	}

	if item.Type != corpus.TypeCode {
		t.Errorf("Type = %q, want %q", item.Type, corpus.TypeCode)
	}
	if item.Metadata["language"] != "go" {
		t.Errorf("language = %q, want 'go'", item.Metadata["language"])
	}

	// Verify in DB
	got, err := db.GetByID(item.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Transcript == "" {
		t.Error("Transcript (code content) should not be empty")
	}
}

func TestIngestCodeDir(t *testing.T) {
	db := testDB(t)
	corpusRoot := t.TempDir()

	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(srcDir, "util.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(srcDir, "readme.md"), []byte("# Readme"), 0644)

	// Create .git dir that should be skipped
	os.MkdirAll(filepath.Join(srcDir, ".git"), 0755)
	os.WriteFile(filepath.Join(srcDir, ".git", "config.go"), []byte("package git"), 0644)

	items, err := IngestCodeDir(db, corpusRoot, srcDir, CodeOptions{Source: "test"})
	if err != nil {
		t.Fatalf("IngestCodeDir: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("IngestCodeDir = %d items, want 2 (should skip .md and .git/)", len(items))
	}
}

func TestIngestCodeDir_FilterByLanguage(t *testing.T) {
	db := testDB(t)
	corpusRoot := t.TempDir()

	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "main.go"), []byte("package main"), 0644)
	os.WriteFile(filepath.Join(srcDir, "app.py"), []byte("print('hello')"), 0644)

	items, err := IngestCodeDir(db, corpusRoot, srcDir, CodeOptions{Language: "go"})
	if err != nil {
		t.Fatalf("IngestCodeDir: %v", err)
	}

	if len(items) != 1 {
		t.Errorf("IngestCodeDir(go) = %d items, want 1", len(items))
	}
}
