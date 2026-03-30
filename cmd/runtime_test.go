package cmd

import (
	"testing"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/tts"
)

func TestInitBackendsUsesConfiguredRuntimePaths(t *testing.T) {
	tts.ClearRegistry()
	cfg := config.DefaultConfig()
	cfg.TTS.Chatterbox.RuntimePath = "/tmp/chatterbox-env"
	cfg.TTS.F5.RuntimePath = "/tmp/f5-env"
	initBackends(cfg)

	b1, err := tts.Get("chatterbox")
	if err != nil {
		t.Fatalf("chatterbox backend missing: %v", err)
	}
	cb, ok := b1.(*tts.ChatterboxBackend)
	if !ok {
		t.Fatalf("unexpected backend type %T", b1)
	}
	if cb.Runtime.PythonPath != "/tmp/chatterbox-env/bin/python3" {
		t.Fatalf("got chatterbox runtime %q", cb.Runtime.PythonPath)
	}

	b2, err := tts.Get("f5-tts")
	if err != nil {
		t.Fatalf("f5 backend missing: %v", err)
	}
	f5, ok := b2.(*tts.F5Backend)
	if !ok {
		t.Fatalf("unexpected backend type %T", b2)
	}
	if f5.Runtime.PythonPath != "/tmp/f5-env/bin/python3" {
		t.Fatalf("got f5 runtime %q", f5.Runtime.PythonPath)
	}
}
