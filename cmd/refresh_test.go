package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cyperx84/voice-forge/internal/config"
)

func TestNeedsRefresh_NoProfile(t *testing.T) {
	should, reason := needsRefresh("/tmp/nonexistent-style.json", 50, config.RefreshConfig{
		MinInterval:       "24h",
		MinNewTranscripts: 20,
	})
	if !should {
		t.Error("should refresh when no profile exists")
	}
	if reason != "no existing profile found" {
		t.Errorf("unexpected reason: %s", reason)
	}
}

func TestNeedsRefresh_RecentWithFewNew(t *testing.T) {
	dir := t.TempDir()
	stylePath := filepath.Join(dir, "style.json")

	profile := map[string]interface{}{
		"analyzed_at":  time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		"sample_count": 100,
	}
	data, _ := json.Marshal(profile)
	os.WriteFile(stylePath, data, 0644)

	should, _ := needsRefresh(stylePath, 105, config.RefreshConfig{
		MinInterval:       "24h",
		MinNewTranscripts: 20,
	})
	if should {
		t.Error("should not refresh when profile is recent and few new transcripts")
	}
}

func TestNeedsRefresh_RecentWithManyNew(t *testing.T) {
	dir := t.TempDir()
	stylePath := filepath.Join(dir, "style.json")

	profile := map[string]interface{}{
		"analyzed_at":  time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
		"sample_count": 100,
	}
	data, _ := json.Marshal(profile)
	os.WriteFile(stylePath, data, 0644)

	should, _ := needsRefresh(stylePath, 125, config.RefreshConfig{
		MinInterval:       "24h",
		MinNewTranscripts: 20,
	})
	if !should {
		t.Error("should refresh when many new transcripts even if recent")
	}
}

func TestNeedsRefresh_OldProfile(t *testing.T) {
	dir := t.TempDir()
	stylePath := filepath.Join(dir, "style.json")

	profile := map[string]interface{}{
		"analyzed_at":  time.Now().Add(-48 * time.Hour).Format(time.RFC3339),
		"sample_count": 100,
	}
	data, _ := json.Marshal(profile)
	os.WriteFile(stylePath, data, 0644)

	should, _ := needsRefresh(stylePath, 115, config.RefreshConfig{
		MinInterval:       "24h",
		MinNewTranscripts: 20,
	})
	if !should {
		t.Error("should refresh when profile is old and has 10%+ growth")
	}
}

func TestNeedsRefresh_NoNewTranscripts(t *testing.T) {
	dir := t.TempDir()
	stylePath := filepath.Join(dir, "style.json")

	profile := map[string]interface{}{
		"analyzed_at":  time.Now().Add(-48 * time.Hour).Format(time.RFC3339),
		"sample_count": 100,
	}
	data, _ := json.Marshal(profile)
	os.WriteFile(stylePath, data, 0644)

	should, _ := needsRefresh(stylePath, 100, config.RefreshConfig{
		MinInterval:       "24h",
		MinNewTranscripts: 20,
	})
	if should {
		t.Error("should not refresh when no new transcripts")
	}
}
