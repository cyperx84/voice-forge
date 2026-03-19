package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cyperx84/voice-forge/internal/config"
)

func TestNeedsRefresh_ZeroSampleCount(t *testing.T) {
	dir := t.TempDir()
	stylePath := filepath.Join(dir, "style.json")

	profile := map[string]interface{}{
		"analyzed_at":  time.Now().Add(-48 * time.Hour).Format(time.RFC3339),
		"sample_count": 0,
	}
	data, _ := json.Marshal(profile)
	os.WriteFile(stylePath, data, 0644)

	should, reason := needsRefresh(stylePath, 10, config.RefreshConfig{
		MinInterval:       "24h",
		MinNewTranscripts: 20,
	})
	if !should {
		t.Error("should refresh when sample_count is 0 to avoid division by zero")
	}
	_ = reason
}

func TestNeedsRefresh_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	stylePath := filepath.Join(dir, "style.json")

	os.WriteFile(stylePath, []byte("not json"), 0644)

	should, reason := needsRefresh(stylePath, 10, config.RefreshConfig{
		MinInterval:       "24h",
		MinNewTranscripts: 20,
	})
	if !should {
		t.Error("should refresh when profile JSON is invalid")
	}
	if reason != "existing profile is invalid" {
		t.Errorf("unexpected reason: %s", reason)
	}
}

func TestNeedsRefresh_InvalidTimestamp(t *testing.T) {
	dir := t.TempDir()
	stylePath := filepath.Join(dir, "style.json")

	profile := map[string]interface{}{
		"analyzed_at":  "not-a-timestamp",
		"sample_count": 100,
	}
	data, _ := json.Marshal(profile)
	os.WriteFile(stylePath, data, 0644)

	should, reason := needsRefresh(stylePath, 110, config.RefreshConfig{
		MinInterval:       "24h",
		MinNewTranscripts: 20,
	})
	if !should {
		t.Error("should refresh when timestamp is invalid")
	}
	if reason != "cannot parse profile timestamp" {
		t.Errorf("unexpected reason: %s", reason)
	}
}
