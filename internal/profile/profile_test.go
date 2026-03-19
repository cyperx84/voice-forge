package profile

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/cyperx84/voice-forge/internal/analyzer"
)

func TestLoad(t *testing.T) {
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "style.json")

	profile := analyzer.StyleProfile{
		AnalyzedAt:      "2026-03-19T12:00:00Z",
		SampleCount:     42,
		TotalWords:      1000,
		OverallVoiceDNA: "Test voice DNA",
		Vocabulary: analyzer.VocabularyProfile{
			PreferredRegister: "casual",
			AvgSentenceLength: 15,
		},
		KeyPhrases: []string{"let's go"},
	}

	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(profilePath, data, 0644); err != nil {
		t.Fatal(err)
	}

	loaded, err := Load(profilePath)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if loaded.SampleCount != 42 {
		t.Errorf("expected SampleCount 42, got %d", loaded.SampleCount)
	}
	if loaded.OverallVoiceDNA != "Test voice DNA" {
		t.Errorf("unexpected voice DNA: %q", loaded.OverallVoiceDNA)
	}
}

func TestLoadNonExistent(t *testing.T) {
	_, err := Load("/nonexistent/style.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "bad.json")
	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"2026-03-19T12:00:00Z", "2026-03-19 12:00"},
		{"invalid", "invalid"},
		{"", ""},
	}

	for _, tt := range tests {
		got := formatTime(tt.input)
		if got != tt.want {
			t.Errorf("formatTime(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestPrintBriefDoesNotPanic(t *testing.T) {
	profile := &analyzer.StyleProfile{
		AnalyzedAt:      "2026-03-19T12:00:00Z",
		SampleCount:     5,
		TotalWords:      100,
		OverallVoiceDNA: "Test",
		KeyPhrases:      []string{"a", "b", "c"},
	}
	// Should not panic
	PrintBrief(profile)
}

func TestPrintFullDoesNotPanic(t *testing.T) {
	profile := &analyzer.StyleProfile{
		AnalyzedAt:      "2026-03-19T12:00:00Z",
		SampleCount:     5,
		TotalWords:      100,
		OverallVoiceDNA: "Test",
		Vocabulary: analyzer.VocabularyProfile{
			CommonSlang:       []string{"yeah"},
			PreferredRegister: "casual",
		},
		Humor: analyzer.HumorProfile{Style: "dry"},
		Rhythm: analyzer.RhythmProfile{
			Patterns: []string{"short-long"},
		},
		KeyPhrases: []string{"go"},
		AvoidList:  []string{"nope"},
	}
	// Should not panic
	PrintFull(profile)
}
