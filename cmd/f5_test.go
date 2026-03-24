package cmd

import (
	"testing"

	"github.com/cyperx84/voice-forge/internal/tts"
)

func TestInitBackendsRegistersF5(t *testing.T) {
	tts.ClearRegistry()
	initBackends(testConfig())
	if _, err := tts.Get("f5-tts"); err != nil {
		t.Fatalf("expected f5-tts backend to be registered: %v", err)
	}
}
