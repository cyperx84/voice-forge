package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cyperx84/voice-forge/internal/analyzer"
)

func mockProfile() *analyzer.StyleProfile {
	return &analyzer.StyleProfile{
		AnalyzedAt:  "2026-03-19T12:00:00Z",
		SampleCount: 50,
		TotalWords:  10000,
		Vocabulary: analyzer.VocabularyProfile{
			CommonSlang:       []string{"bro", "nah", "lowkey"},
			Contractions:      []string{"don't", "can't", "I'm"},
			FillerWords:       []string{"like", "you know", "basically"},
			TechnicalTerms:    []string{"API", "backend", "deployment"},
			AvgSentenceLength: 12,
			PreferredRegister: "casual-technical",
		},
		Humor: analyzer.HumorProfile{
			Style:       "dry, self-deprecating",
			Frequency:   "frequent",
			Examples:    []string{"yeah that's totally not gonna blow up in prod"},
			Description: "Uses humor to soften technical criticism",
		},
		ArgumentStyle: analyzer.ArgumentProfile{
			Pattern:     "example-first, then principle",
			Transitions: []string{"so basically", "the thing is", "look"},
			Persuasion:  "shows rather than tells",
			Description: "Leads with concrete examples before abstracting",
		},
		EmotionalRange: analyzer.EmotionalProfile{
			PrimaryTones: []string{"enthusiastic", "frustrated", "curious"},
			Triggers:     []string{"bad code", "cool tech", "bureaucracy"},
			Intensity:    "moderate-high",
			Description:  "Passionate about craft, impatient with waste",
		},
		Rhythm: analyzer.RhythmProfile{
			SentenceVariation: "high",
			Pacing:            "variable — fast when excited, measured when explaining",
			Patterns:          []string{"short setup → punchy line → detail"},
			Description:       "Mixes rapid-fire points with longer explanations",
		},
		KeyPhrases:      []string{"here's the thing", "no cap", "that's the play"},
		AvoidList:        []string{"utilize", "synergy", "circle back", "leverage"},
		OverallVoiceDNA: "A technical builder who speaks in examples and ships fast. Direct, casual, occasionally funny.",
	}
}

func TestGenerate(t *testing.T) {
	dir := t.TempDir()
	p := mockProfile()

	if err := Generate(p, dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Check all files exist
	expectedFiles := []string{
		"SKILL.md",
		"references/voice-profile.md",
		"references/avoid-list.md",
		"references/key-phrases.md",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(dir, f)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("expected file %s to exist", f)
		}
	}
}

func TestSkillMdContent(t *testing.T) {
	dir := t.TempDir()
	p := mockProfile()

	if err := Generate(p, dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "SKILL.md"))
	if err != nil {
		t.Fatalf("reading SKILL.md: %v", err)
	}
	content := string(data)

	checks := []string{
		"cyperx-voice",
		"references/voice-profile.md",
		"references/key-phrases.md",
		"references/avoid-list.md",
		"Voice DNA",
		p.OverallVoiceDNA,
		"casual-technical",
	}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("SKILL.md should contain %q", check)
		}
	}
}

func TestAvoidListContent(t *testing.T) {
	dir := t.TempDir()
	p := mockProfile()

	if err := Generate(p, dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "references/avoid-list.md"))
	if err != nil {
		t.Fatalf("reading avoid-list.md: %v", err)
	}
	content := string(data)

	for _, word := range p.AvoidList {
		if !strings.Contains(content, word) {
			t.Errorf("avoid-list.md should contain %q", word)
		}
	}
}

func TestKeyPhrasesContent(t *testing.T) {
	dir := t.TempDir()
	p := mockProfile()

	if err := Generate(p, dir); err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "references/key-phrases.md"))
	if err != nil {
		t.Fatalf("reading key-phrases.md: %v", err)
	}
	content := string(data)

	for _, phrase := range p.KeyPhrases {
		if !strings.Contains(content, phrase) {
			t.Errorf("key-phrases.md should contain %q", phrase)
		}
	}
}

func TestGenerateEmptyProfile(t *testing.T) {
	dir := t.TempDir()
	p := &analyzer.StyleProfile{}

	if err := Generate(p, dir); err != nil {
		t.Fatalf("Generate with empty profile failed: %v", err)
	}

	// Avoid list should show "no entries" message
	data, _ := os.ReadFile(filepath.Join(dir, "references/avoid-list.md"))
	if !strings.Contains(string(data), "No avoid list entries yet") {
		t.Error("empty avoid list should have placeholder message")
	}
}
