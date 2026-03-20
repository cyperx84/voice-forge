package ingest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseTwitterArchive_Wrapped(t *testing.T) {
	tweets := []Tweet{
		{Tweet: struct {
			ID        string `json:"id_str"`
			FullText  string `json:"full_text"`
			CreatedAt string `json:"created_at"`
		}{ID: "1", FullText: "Just shipped a new feature!", CreatedAt: "2026-01-15T10:00:00Z"}},
		{Tweet: struct {
			ID        string `json:"id_str"`
			FullText  string `json:"full_text"`
			CreatedAt string `json:"created_at"`
		}{ID: "2", FullText: "Coding all night", CreatedAt: "2026-01-15T22:00:00Z"}},
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "tweets.json")
	data, _ := json.Marshal(tweets)
	os.WriteFile(path, data, 0644)

	posts, err := ParseTwitterArchive(path)
	if err != nil {
		t.Fatalf("ParseTwitterArchive: %v", err)
	}

	if len(posts) != 2 {
		t.Errorf("got %d posts, want 2", len(posts))
	}
	if posts[0].Text != "Just shipped a new feature!" {
		t.Errorf("posts[0].Text = %q", posts[0].Text)
	}
	if posts[0].Source != "twitter" {
		t.Errorf("posts[0].Source = %q", posts[0].Source)
	}
}

func TestParseTwitterArchive_Simple(t *testing.T) {
	// Use the simple format (array of flat tweet objects)
	raw := `[{"id_str":"1","full_text":"Hello from Twitter","created_at":"2026-01-15T10:00:00Z"},{"id_str":"2","text":"Fallback text field","created_at":"2026-01-15T11:00:00Z"}]`

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "tweets.json")
	os.WriteFile(path, []byte(raw), 0644)

	posts, err := ParseTwitterArchive(path)
	if err != nil {
		t.Fatalf("ParseTwitterArchive: %v", err)
	}

	if len(posts) != 2 {
		t.Errorf("got %d posts, want 2", len(posts))
	}
}
