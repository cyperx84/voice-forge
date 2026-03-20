package ingest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cyperx84/voice-forge/internal/corpus"
)

func TestIsPhotoFile(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"photo.jpg", true},
		{"photo.JPEG", true},
		{"photo.png", true},
		{"photo.webp", true},
		{"photo.gif", true},
		{"photo.txt", false},
		{"photo.go", false},
	}
	for _, tt := range tests {
		got := IsPhotoFile(tt.name)
		if got != tt.want {
			t.Errorf("IsPhotoFile(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestIngestPhotoFile(t *testing.T) {
	db := testDB(t)
	corpusRoot := t.TempDir()

	srcDir := t.TempDir()
	srcPath := filepath.Join(srcDir, "test.jpg")
	// Write fake JPEG data (just needs to be a file)
	os.WriteFile(srcPath, []byte("fake jpeg data"), 0644)

	item, err := IngestPhotoFile(db, corpusRoot, srcPath, PhotoOptions{
		Source: "brand-kit",
		Tags:   []string{"profile", "headshot"},
	})
	if err != nil {
		t.Fatalf("IngestPhotoFile: %v", err)
	}

	if item.Type != corpus.TypePhoto {
		t.Errorf("Type = %q, want %q", item.Type, corpus.TypePhoto)
	}
	if item.Source != "brand-kit" {
		t.Errorf("Source = %q", item.Source)
	}

	// Verify file was copied
	destPath := filepath.Join(corpusRoot, item.Path)
	if _, err := os.Stat(destPath); err != nil {
		t.Errorf("copied file not found: %v", err)
	}
}

func TestIngestPhotoDir(t *testing.T) {
	db := testDB(t)
	corpusRoot := t.TempDir()

	srcDir := t.TempDir()
	os.WriteFile(filepath.Join(srcDir, "a.jpg"), []byte("fake"), 0644)
	os.WriteFile(filepath.Join(srcDir, "b.png"), []byte("fake"), 0644)
	os.WriteFile(filepath.Join(srcDir, "c.txt"), []byte("not a photo"), 0644)

	items, err := IngestPhotoDir(db, corpusRoot, srcDir, PhotoOptions{Source: "test"})
	if err != nil {
		t.Fatalf("IngestPhotoDir: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("IngestPhotoDir = %d items, want 2", len(items))
	}
}

func TestIngestPhotoDir_Recursive(t *testing.T) {
	db := testDB(t)
	corpusRoot := t.TempDir()

	srcDir := t.TempDir()
	subDir := filepath.Join(srcDir, "sub")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(srcDir, "a.jpg"), []byte("fake"), 0644)
	os.WriteFile(filepath.Join(subDir, "b.jpg"), []byte("fake"), 0644)

	// Without recursive — should only get top-level
	items, _ := IngestPhotoDir(db, corpusRoot, srcDir, PhotoOptions{Recursive: false})
	if len(items) != 1 {
		t.Errorf("non-recursive = %d items, want 1", len(items))
	}

	// With recursive — should get both
	db2 := testDB(t)
	items, _ = IngestPhotoDir(db2, corpusRoot, srcDir, PhotoOptions{Recursive: true})
	if len(items) != 2 {
		t.Errorf("recursive = %d items, want 2", len(items))
	}
}
