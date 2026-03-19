package rewriter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cyperx84/voice-forge/internal/character"
)

func TestLoadStyleJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "style.json")

	validJSON := `{"overall_voice_dna": "test voice", "vocabulary": {}}`
	if err := os.WriteFile(path, []byte(validJSON), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := LoadStyleJSON(path)
	if err != nil {
		t.Fatalf("LoadStyleJSON failed: %v", err)
	}

	if got != validJSON {
		t.Errorf("LoadStyleJSON = %q, want %q", got, validJSON)
	}
}

func TestLoadStyleJSONInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "style.json")

	if err := os.WriteFile(path, []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadStyleJSON(path)
	if err == nil {
		t.Error("LoadStyleJSON should fail for invalid JSON")
	}
}

func TestLoadStyleJSONMissing(t *testing.T) {
	_, err := LoadStyleJSON("/nonexistent/style.json")
	if err == nil {
		t.Error("LoadStyleJSON should fail for missing file")
	}
}

func TestRewriteWithEcho(t *testing.T) {
	ch := &character.Character{
		Name:        "test",
		Description: "test character",
		ToneShift: character.ToneShift{
			Register:   "formal",
			Pacing:     "slow",
			Persona:    "test persona",
			EmojiStyle: "none",
			Vocabulary: []string{"indeed"},
			AvoidWords: []string{"yo"},
		},
	}

	// Use echo as a mock LLM — it will echo stdin back (but actually cat)
	result, err := Rewrite("hello world", ch, "{}", "cat", nil)
	if err != nil {
		t.Fatalf("Rewrite failed: %v", err)
	}

	// cat reads stdin, so result should contain our prompt text
	if result == "" {
		t.Error("Rewrite returned empty string")
	}
}

func TestGenerateWithEcho(t *testing.T) {
	ch := &character.Character{
		Name:        "test",
		Description: "test character",
		ToneShift: character.ToneShift{
			Register:   "casual",
			Pacing:     "fast",
			Persona:    "test persona",
			EmojiStyle: "minimal",
		},
	}

	result, err := Generate("test topic", ch, "{}", "cat", nil)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result == "" {
		t.Error("Generate returned empty string")
	}
}
