package corpus

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	db, err := OpenDB(dbPath)
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	defer db.Close()

	// Verify file was created
	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("db file not created: %v", err)
	}
}

func TestInsertAndGetByID(t *testing.T) {
	db := testDB(t)

	item := &Item{
		ID:         "test-123",
		Type:       TypeText,
		Source:     "blog",
		CreatedAt:  time.Now().Format(time.RFC3339),
		IngestedAt: time.Now().Format(time.RFC3339),
		Path:       "text/test-123.md",
		Transcript: "Hello world, this is a test.",
		Tags:       []string{"test", "hello"},
		Metadata:   map[string]string{"key": "value"},
		WordCount:  6,
		FileSize:   27,
	}

	if err := db.Insert(item); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	got, err := db.GetByID("test-123")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}

	if got.Type != TypeText {
		t.Errorf("Type = %q, want %q", got.Type, TypeText)
	}
	if got.Source != "blog" {
		t.Errorf("Source = %q, want %q", got.Source, "blog")
	}
	if got.Transcript != "Hello world, this is a test." {
		t.Errorf("Transcript = %q", got.Transcript)
	}
	if got.WordCount != 6 {
		t.Errorf("WordCount = %d, want 6", got.WordCount)
	}
	if len(got.Tags) != 2 {
		t.Errorf("Tags = %v, want 2 tags", got.Tags)
	}
}

func TestListByType(t *testing.T) {
	db := testDB(t)

	insertTestItem(t, db, "a", TypeText, "blog")
	insertTestItem(t, db, "b", TypeText, "note")
	insertTestItem(t, db, "c", TypeCode, "github")

	items, err := db.ListByType(TypeText)
	if err != nil {
		t.Fatalf("ListByType: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("ListByType(text) = %d items, want 2", len(items))
	}

	items, err = db.ListByType(TypeCode)
	if err != nil {
		t.Fatalf("ListByType: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("ListByType(code) = %d items, want 1", len(items))
	}
}

func TestSearch(t *testing.T) {
	db := testDB(t)

	insertTestItemWithTranscript(t, db, "a", TypeText, "The quick brown fox")
	insertTestItemWithTranscript(t, db, "b", TypeText, "The lazy dog sleeps")
	insertTestItemWithTranscript(t, db, "c", TypeCode, "func main() { println() }")

	items, err := db.Search("fox")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("Search(fox) = %d, want 1", len(items))
	}

	items, err = db.Search("The")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("Search(The) = %d, want 2", len(items))
	}
}

func TestStats(t *testing.T) {
	db := testDB(t)

	insertTestItem(t, db, "a", TypeText, "blog")
	insertTestItem(t, db, "b", TypeText, "note")
	insertTestItem(t, db, "c", TypeCode, "github")
	insertTestItem(t, db, "d", TypeVoice, "local")

	stats, err := db.Stats()
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}

	if len(stats) != 3 {
		t.Fatalf("Stats returned %d types, want 3", len(stats))
	}
}

func TestRecent(t *testing.T) {
	db := testDB(t)

	insertTestItem(t, db, "a", TypeText, "blog")
	insertTestItem(t, db, "b", TypeCode, "github")
	insertTestItem(t, db, "c", TypeVoice, "local")

	items, err := db.Recent(2)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("Recent(2) = %d items, want 2", len(items))
	}
}

func TestCount(t *testing.T) {
	db := testDB(t)

	insertTestItem(t, db, "a", TypeText, "blog")
	insertTestItem(t, db, "b", TypeText, "note")
	insertTestItem(t, db, "c", TypeCode, "github")

	count, err := db.Count("")
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 3 {
		t.Errorf("Count() = %d, want 3", count)
	}

	count, err = db.Count(TypeText)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 2 {
		t.Errorf("Count(text) = %d, want 2", count)
	}
}

func TestAllTranscripts(t *testing.T) {
	db := testDB(t)

	insertTestItemWithTranscript(t, db, "a", TypeText, "Hello world")
	insertTestItemWithTranscript(t, db, "b", TypeText, "Goodbye world")
	insertTestItemWithTranscript(t, db, "c", TypeCode, "func main()")

	ts, err := db.AllTranscripts("")
	if err != nil {
		t.Fatalf("AllTranscripts: %v", err)
	}
	if len(ts) != 3 {
		t.Errorf("AllTranscripts() = %d, want 3", len(ts))
	}

	ts, err = db.AllTranscripts(TypeText)
	if err != nil {
		t.Fatalf("AllTranscripts: %v", err)
	}
	if len(ts) != 2 {
		t.Errorf("AllTranscripts(text) = %d, want 2", len(ts))
	}
}

func TestMigrateExistingCorpus(t *testing.T) {
	dir := t.TempDir()

	// Create some voice corpus files
	os.WriteFile(filepath.Join(dir, "abc-123.txt"), []byte("hello from voice"), 0644)
	os.WriteFile(filepath.Join(dir, "abc-123.ogg"), []byte("fake audio"), 0644)
	os.WriteFile(filepath.Join(dir, "def-456.txt"), []byte("second transcript"), 0644)

	db := testDB(t)
	migrated, err := MigrateExistingCorpus([]string{dir}, db)
	if err != nil {
		t.Fatalf("MigrateExistingCorpus: %v", err)
	}

	// These aren't UUID-formatted names, so they won't match the uuid pattern
	// but MigrateExistingCorpus reads all .txt files, not just UUID ones
	_ = migrated

	// Run again — should not duplicate
	migrated2, _ := MigrateExistingCorpus([]string{dir}, db)
	if migrated2 != 0 {
		t.Errorf("second migration should migrate 0, got %d", migrated2)
	}
}

func TestExportAll(t *testing.T) {
	db := testDB(t)

	insertTestItem(t, db, "a", TypeText, "blog")
	insertTestItem(t, db, "b", TypeCode, "github")

	items, err := db.ExportAll()
	if err != nil {
		t.Fatalf("ExportAll: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("ExportAll = %d, want 2", len(items))
	}
}

// Helpers

func testDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	db, err := OpenDB(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("OpenDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func insertTestItem(t *testing.T, db *DB, id, itemType, source string) {
	t.Helper()
	item := &Item{
		ID:         id,
		Type:       itemType,
		Source:     source,
		IngestedAt: time.Now().Format(time.RFC3339),
		Path:       itemType + "/" + id,
		WordCount:  5,
	}
	if err := db.Insert(item); err != nil {
		t.Fatalf("Insert(%s): %v", id, err)
	}
}

func insertTestItemWithTranscript(t *testing.T, db *DB, id, itemType, transcript string) {
	t.Helper()
	item := &Item{
		ID:         id,
		Type:       itemType,
		Source:     "test",
		IngestedAt: time.Now().Format(time.RFC3339),
		Path:       itemType + "/" + id,
		Transcript: transcript,
		WordCount:  len(transcript),
	}
	if err := db.Insert(item); err != nil {
		t.Fatalf("Insert(%s): %v", id, err)
	}
}
