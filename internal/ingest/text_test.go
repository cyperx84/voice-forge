package ingest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cyperx84/voice-forge/internal/corpus"
)

func TestIngestTextFile(t *testing.T) {
	db := testDB(t)
	corpusRoot := t.TempDir()

	// Create a test file
	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "post.md")
	os.WriteFile(srcPath, []byte("# My Blog Post\n\nThis is a test blog post with some content."), 0644)

	item, err := IngestTextFile(db, corpusRoot, srcPath, TextOptions{
		Source: "blog",
		Tags:   []string{"test"},
	})
	if err != nil {
		t.Fatalf("IngestTextFile: %v", err)
	}

	if item.Type != corpus.TypeText {
		t.Errorf("Type = %q, want %q", item.Type, corpus.TypeText)
	}
	if item.Source != "blog" {
		t.Errorf("Source = %q, want %q", item.Source, "blog")
	}
	if item.WordCount == 0 {
		t.Error("WordCount should be > 0")
	}

	// Verify file was copied
	destPath := filepath.Join(corpusRoot, item.Path)
	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("copied file not found: %v", err)
	}

	// Verify in DB
	got, err := db.GetByID(item.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Transcript == "" {
		t.Error("Transcript should not be empty")
	}
}

func TestIngestTextFile_Empty(t *testing.T) {
	db := testDB(t)
	corpusRoot := t.TempDir()

	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "empty.md")
	os.WriteFile(srcPath, []byte(""), 0644)

	_, err := IngestTextFile(db, corpusRoot, srcPath, TextOptions{})
	if err == nil {
		t.Error("expected error for empty file")
	}
}

func TestIngestTextString(t *testing.T) {
	db := testDB(t)
	corpusRoot := t.TempDir()

	item, err := IngestTextString(db, corpusRoot, "Hello world this is a test", "note", nil)
	if err != nil {
		t.Fatalf("IngestTextString: %v", err)
	}

	if item.Type != corpus.TypeText {
		t.Errorf("Type = %q", item.Type)
	}
	if item.WordCount != 6 {
		t.Errorf("WordCount = %d, want 6", item.WordCount)
	}
}

func TestIngestTextString_Empty(t *testing.T) {
	db := testDB(t)
	corpusRoot := t.TempDir()

	_, err := IngestTextString(db, corpusRoot, "  ", "note", nil)
	if err == nil {
		t.Error("expected error for empty text")
	}
}

func testDB(t *testing.T) *corpus.DB {
	t.Helper()
	dir := t.TempDir()
	db, err := corpus.OpenDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}
