package ingest

import (
	"testing"

	"github.com/cyperx84/voice-forge/internal/corpus"
)

func TestIngestSocialPosts(t *testing.T) {
	db := testDB(t)
	corpusRoot := t.TempDir()

	posts := []SocialPost{
		{Text: "Just shipped a new feature!", Author: "cyperx", Source: "twitter", Timestamp: "2026-01-15T10:00:00Z"},
		{Text: "Coding all night long", Author: "cyperx", Source: "twitter", Timestamp: "2026-01-15T22:00:00Z"},
		{Text: "", Author: "cyperx"}, // empty, should be skipped
	}

	items, err := IngestSocialPosts(db, corpusRoot, posts, "twitter")
	if err != nil {
		t.Fatalf("IngestSocialPosts: %v", err)
	}

	if len(items) != 2 {
		t.Errorf("got %d items, want 2", len(items))
	}

	if items[0].Type != corpus.TypeSocial {
		t.Errorf("Type = %q, want %q", items[0].Type, corpus.TypeSocial)
	}

	// Verify in DB
	count, _ := db.Count(corpus.TypeSocial)
	if count != 2 {
		t.Errorf("DB count = %d, want 2", count)
	}
}
