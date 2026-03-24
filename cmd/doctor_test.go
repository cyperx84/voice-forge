package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cyperx84/voice-forge/internal/config"
)

func TestCorpusFootprintCountsDBManagedAndSources(t *testing.T) {
	tmp := t.TempDir()
	dbPath := filepath.Join(tmp, "corpus.db")
	managedDir := filepath.Join(tmp, "managed")
	sourceDir := filepath.Join(tmp, "source")

	if err := os.WriteFile(dbPath, []byte("db-bytes"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(managedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(managedDir, "a.txt"), []byte("managed"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDir, "b.txt"), []byte("source"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{}
	cfg.Corpus.DB = dbPath
	cfg.Corpus.Root = managedDir
	cfg.Corpus.Paths = []string{sourceDir, sourceDir}

	fp := corpusFootprint(cfg)
	if fp.db <= 0 {
		t.Fatalf("expected db footprint > 0, got %d", fp.db)
	}
	if fp.managed <= 0 {
		t.Fatalf("expected managed footprint > 0, got %d", fp.managed)
	}
	if fp.sources <= 0 {
		t.Fatalf("expected source footprint > 0, got %d", fp.sources)
	}
	if fp.total() != fp.db+fp.managed+fp.sources {
		t.Fatalf("unexpected total: %d", fp.total())
	}
}
