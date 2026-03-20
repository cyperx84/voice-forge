package ingest

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseDiscordExport_Array(t *testing.T) {
	messages := []DiscordMessage{
		{ID: "1", Content: "Hello everyone!", Timestamp: "2026-01-15T10:00:00Z", Author: struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		}{Name: "cyperx", ID: "123"}},
		{ID: "2", Content: "How's it going?", Timestamp: "2026-01-15T10:01:00Z", Author: struct {
			Name string `json:"name"`
			ID   string `json:"id"`
		}{Name: "cyperx", ID: "123"}},
		{ID: "3", Content: "", Timestamp: "2026-01-15T10:02:00Z"}, // empty, should be skipped
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "discord.json")
	data, _ := json.Marshal(messages)
	os.WriteFile(path, data, 0644)

	posts, err := ParseDiscordExport(path)
	if err != nil {
		t.Fatalf("ParseDiscordExport: %v", err)
	}

	if len(posts) != 2 {
		t.Errorf("got %d posts, want 2", len(posts))
	}
	if posts[0].Text != "Hello everyone!" {
		t.Errorf("posts[0].Text = %q", posts[0].Text)
	}
	if posts[0].Source != "discord" {
		t.Errorf("posts[0].Source = %q", posts[0].Source)
	}
}

func TestParseDiscordExport_Object(t *testing.T) {
	export := DiscordExport{
		Messages: []DiscordMessage{
			{ID: "1", Content: "Test message", Timestamp: "2026-01-15T10:00:00Z"},
		},
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "discord.json")
	data, _ := json.Marshal(export)
	os.WriteFile(path, data, 0644)

	posts, err := ParseDiscordExport(path)
	if err != nil {
		t.Fatalf("ParseDiscordExport: %v", err)
	}

	if len(posts) != 1 {
		t.Errorf("got %d posts, want 1", len(posts))
	}
}
