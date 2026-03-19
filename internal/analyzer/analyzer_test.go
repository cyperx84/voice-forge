package analyzer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestStripCodeFences(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`{"key": "value"}`, `{"key": "value"}`},
		{"```json\n{\"key\": \"value\"}\n```", `{"key": "value"}`},
		{"```\n{\"key\": \"value\"}\n```", `{"key": "value"}`},
		{"  ```json\n{}\n```  ", `{}`},
	}

	for _, tt := range tests {
		got := stripCodeFences(tt.input)
		if got != tt.want {
			t.Errorf("stripCodeFences(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSaveProfile(t *testing.T) {
	tmpDir := t.TempDir()

	profile := &StyleProfile{
		AnalyzedAt:  "2026-03-19T12:00:00Z",
		SampleCount: 10,
		TotalWords:  500,
		Vocabulary: VocabularyProfile{
			CommonSlang:       []string{"yeah", "gonna"},
			Contractions:      []string{"don't", "it's"},
			FillerWords:       []string{"like", "you know"},
			TechnicalTerms:    []string{"API", "backend"},
			AvgSentenceLength: 12.5,
			PreferredRegister: "casual",
		},
		Humor: HumorProfile{
			Style:       "dry",
			Frequency:   "moderate",
			Examples:    []string{"test joke"},
			Description: "Dry humor with tech references",
		},
		ArgumentStyle: ArgumentProfile{
			Pattern:     "direct",
			Transitions: []string{"so", "but"},
			Persuasion:  "evidence-based",
			Description: "Gets to the point quickly",
		},
		EmotionalRange: EmotionalProfile{
			PrimaryTones: []string{"enthusiastic", "pragmatic"},
			Triggers:     []string{"tech progress"},
			Intensity:    "moderate",
			Description:  "Generally positive",
		},
		Rhythm: RhythmProfile{
			SentenceVariation: "high",
			Pacing:            "fast",
			Patterns:          []string{"short setup, detail"},
			Description:       "Quick pacing",
		},
		KeyPhrases:      []string{"let's go", "ship it"},
		AvoidList:        []string{"hereby", "furthermore"},
		OverallVoiceDNA: "Direct, casual, tech-savvy speaker.",
	}

	if err := SaveProfile(profile, tmpDir); err != nil {
		t.Fatalf("SaveProfile() error: %v", err)
	}

	// Verify style.json exists and is valid JSON
	jsonPath := filepath.Join(tmpDir, "style.json")
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read style.json: %v", err)
	}

	var loaded StyleProfile
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("style.json is not valid JSON: %v", err)
	}

	if loaded.SampleCount != 10 {
		t.Errorf("expected SampleCount 10, got %d", loaded.SampleCount)
	}
	if loaded.OverallVoiceDNA != "Direct, casual, tech-savvy speaker." {
		t.Errorf("unexpected voice DNA: %q", loaded.OverallVoiceDNA)
	}

	// Verify style-summary.md exists
	mdPath := filepath.Join(tmpDir, "style-summary.md")
	mdData, err := os.ReadFile(mdPath)
	if err != nil {
		t.Fatalf("failed to read style-summary.md: %v", err)
	}
	if len(mdData) == 0 {
		t.Error("style-summary.md is empty")
	}
}

func TestGenerateSummary(t *testing.T) {
	profile := &StyleProfile{
		AnalyzedAt:      "2026-03-19T12:00:00Z",
		SampleCount:     5,
		TotalWords:      100,
		OverallVoiceDNA: "Test voice",
		Vocabulary: VocabularyProfile{
			PreferredRegister: "casual",
			AvgSentenceLength: 10,
			CommonSlang:       []string{"yeah"},
		},
		KeyPhrases: []string{"let's go"},
		AvoidList:  []string{"moreover"},
	}

	summary := generateSummary(profile)
	if summary == "" {
		t.Error("expected non-empty summary")
	}
	if len(summary) < 50 {
		t.Error("summary seems too short")
	}
}

func TestStyleProfileJSON(t *testing.T) {
	profile := StyleProfile{
		AnalyzedAt:      "2026-03-19T12:00:00Z",
		SampleCount:     1,
		TotalWords:      10,
		OverallVoiceDNA: "test",
	}

	data, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var loaded StyleProfile
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if loaded.AnalyzedAt != profile.AnalyzedAt {
		t.Errorf("roundtrip mismatch: %q vs %q", loaded.AnalyzedAt, profile.AnalyzedAt)
	}
}
