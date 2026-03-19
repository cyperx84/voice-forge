package tts

import (
	"testing"
)

func TestRegisterAndGet(t *testing.T) {
	// Clear registry for test isolation
	registryMu.Lock()
	registry = map[string]Backend{}
	registryMu.Unlock()

	mock := &mockBackend{name: "test-backend"}
	Register(mock)

	b, err := Get("test-backend")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if b.Name() != "test-backend" {
		t.Errorf("got name %q, want %q", b.Name(), "test-backend")
	}
}

func TestGetUnknownBackend(t *testing.T) {
	registryMu.Lock()
	registry = map[string]Backend{}
	registryMu.Unlock()

	_, err := Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown backend")
	}
}

func TestNames(t *testing.T) {
	registryMu.Lock()
	registry = map[string]Backend{}
	registryMu.Unlock()

	Register(&mockBackend{name: "alpha"})
	Register(&mockBackend{name: "beta"})

	names := Names()
	if len(names) != 2 {
		t.Fatalf("got %d names, want 2", len(names))
	}
}

func TestToolkitAvailable(t *testing.T) {
	tk := &ToolkitBackend{Path: "/nonexistent/path"}
	if tk.Available() {
		t.Error("toolkit should not be available for nonexistent path")
	}

	tk2 := &ToolkitBackend{Path: ""}
	if tk2.Available() {
		t.Error("toolkit should not be available with empty path")
	}
}

func TestElevenLabsAvailable(t *testing.T) {
	el := &ElevenLabsBackend{APIKey: ""}
	if el.Available() {
		t.Error("elevenlabs should not be available without API key")
	}

	el2 := &ElevenLabsBackend{APIKey: "sk-test"}
	if !el2.Available() {
		t.Error("elevenlabs should be available with API key")
	}
}

func TestKokoroFallback(t *testing.T) {
	k := &KokoroBackend{Toolkit: nil}
	if k.Available() {
		t.Error("kokoro should not be available without toolkit")
	}

	k2 := &KokoroBackend{Toolkit: &ToolkitBackend{Path: "/nonexistent"}}
	if k2.Available() {
		t.Error("kokoro should not be available when toolkit path doesn't exist")
	}
}

func TestSpeakOptsDefaults(t *testing.T) {
	opts := SpeakOpts{}
	if opts.Voice != "" {
		t.Error("default voice should be empty")
	}
	if opts.Speed != 0 {
		t.Error("default speed should be 0")
	}
	if opts.Format != "" {
		t.Error("default format should be empty")
	}
}

func TestMockBackendSpeak(t *testing.T) {
	mock := &mockBackend{
		name:      "mock",
		available: true,
		audio:     []byte("fake-audio-data"),
	}
	Register(mock)

	audio, err := mock.Speak("hello", SpeakOpts{})
	if err != nil {
		t.Fatalf("speak error: %v", err)
	}
	if string(audio) != "fake-audio-data" {
		t.Errorf("got %q, want %q", audio, "fake-audio-data")
	}
}

func TestMockBackendClone(t *testing.T) {
	mock := &mockBackend{
		name:      "mock",
		available: true,
	}

	err := mock.Clone([]string{"/tmp/sample1.wav", "/tmp/sample2.wav"}, "testvoice")
	if err != nil {
		t.Fatalf("clone error: %v", err)
	}
	if mock.clonedName != "testvoice" {
		t.Errorf("cloned name = %q, want %q", mock.clonedName, "testvoice")
	}
	if len(mock.clonedSamples) != 2 {
		t.Errorf("cloned %d samples, want 2", len(mock.clonedSamples))
	}
}

// mockBackend implements Backend for testing
type mockBackend struct {
	name           string
	available      bool
	audio          []byte
	clonedName     string
	clonedSamples  []string
}

func (m *mockBackend) Name() string     { return m.name }
func (m *mockBackend) Available() bool   { return m.available }
func (m *mockBackend) Setup() error      { return nil }

func (m *mockBackend) Speak(text string, opts SpeakOpts) ([]byte, error) {
	return m.audio, nil
}

func (m *mockBackend) Clone(samples []string, name string) error {
	m.clonedName = name
	m.clonedSamples = samples
	return nil
}
