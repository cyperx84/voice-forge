package character

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()

	ch := &Character{
		Name:        "test-char",
		Description: "A test character",
		BasedOn:     "cyperx",
		ToneShift: ToneShift{
			Register:   "formal",
			Pacing:     "slow",
			Vocabulary: []string{"Indeed", "Furthermore"},
			AvoidWords: []string{"yo", "dude"},
			Persona:    "Test persona",
			EmojiStyle: "none",
		},
		VoiceOpts: map[string]string{
			"voice": "test-voice",
		},
	}

	if err := Save(ch, dir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	path := filepath.Join(dir, "test-char.toml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("character file was not created")
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Name != ch.Name {
		t.Errorf("Name = %q, want %q", loaded.Name, ch.Name)
	}
	if loaded.Description != ch.Description {
		t.Errorf("Description = %q, want %q", loaded.Description, ch.Description)
	}
	if loaded.BasedOn != ch.BasedOn {
		t.Errorf("BasedOn = %q, want %q", loaded.BasedOn, ch.BasedOn)
	}
	if loaded.ToneShift.Register != ch.ToneShift.Register {
		t.Errorf("Register = %q, want %q", loaded.ToneShift.Register, ch.ToneShift.Register)
	}
	if loaded.ToneShift.Pacing != ch.ToneShift.Pacing {
		t.Errorf("Pacing = %q, want %q", loaded.ToneShift.Pacing, ch.ToneShift.Pacing)
	}
	if loaded.ToneShift.Persona != ch.ToneShift.Persona {
		t.Errorf("Persona = %q, want %q", loaded.ToneShift.Persona, ch.ToneShift.Persona)
	}
	if len(loaded.ToneShift.Vocabulary) != 2 {
		t.Errorf("Vocabulary length = %d, want 2", len(loaded.ToneShift.Vocabulary))
	}
	if len(loaded.ToneShift.AvoidWords) != 2 {
		t.Errorf("AvoidWords length = %d, want 2", len(loaded.ToneShift.AvoidWords))
	}
	if loaded.VoiceOpts["voice"] != "test-voice" {
		t.Errorf("VoiceOpts[voice] = %q, want %q", loaded.VoiceOpts["voice"], "test-voice")
	}
}

func TestEnsurePresets(t *testing.T) {
	dir := t.TempDir()

	if err := EnsurePresets(dir); err != nil {
		t.Fatalf("EnsurePresets failed: %v", err)
	}

	// Check all 4 presets were created
	expected := []string{"narrator.toml", "podcast-host.toml", "storyteller.toml", "hype.toml"}
	for _, name := range expected {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("preset %q was not created", name)
		}
	}

	// Verify a preset loads correctly
	ch, err := Load(filepath.Join(dir, "narrator.toml"))
	if err != nil {
		t.Fatalf("loading narrator preset: %v", err)
	}
	if ch.Name != "narrator" {
		t.Errorf("Name = %q, want %q", ch.Name, "narrator")
	}
	if ch.ToneShift.Register != "formal" {
		t.Errorf("Register = %q, want %q", ch.ToneShift.Register, "formal")
	}
}

func TestEnsurePresetsIdempotent(t *testing.T) {
	dir := t.TempDir()

	if err := EnsurePresets(dir); err != nil {
		t.Fatalf("first EnsurePresets failed: %v", err)
	}

	// Modify one preset
	ch, _ := Load(filepath.Join(dir, "narrator.toml"))
	ch.Description = "modified"
	Save(ch, dir)

	// Run again — should not overwrite
	if err := EnsurePresets(dir); err != nil {
		t.Fatalf("second EnsurePresets failed: %v", err)
	}

	ch2, _ := Load(filepath.Join(dir, "narrator.toml"))
	if ch2.Description != "modified" {
		t.Error("EnsurePresets overwrote existing preset")
	}
}

func TestList(t *testing.T) {
	dir := t.TempDir()

	chars, err := List(dir)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	// Should have 4 built-in presets
	if len(chars) != 4 {
		t.Errorf("List returned %d characters, want 4", len(chars))
	}
}

func TestGet(t *testing.T) {
	dir := t.TempDir()

	ch, err := Get("narrator", dir)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if ch.Name != "narrator" {
		t.Errorf("Name = %q, want %q", ch.Name, "narrator")
	}
}

func TestGetNotFound(t *testing.T) {
	dir := t.TempDir()

	_, err := Get("nonexistent", dir)
	if err == nil {
		t.Error("Get should fail for nonexistent character")
	}
}

func TestLoadInvalidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	os.WriteFile(path, []byte("this is not valid toml {{{{"), 0644)

	_, err := Load(path)
	if err == nil {
		t.Error("Load should fail for invalid TOML")
	}
}
