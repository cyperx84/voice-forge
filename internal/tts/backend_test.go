package tts

import (
	"testing"
)

func TestRegisterDuplicate(t *testing.T) {
	registryMu.Lock()
	registry = map[string]Backend{}
	registryMu.Unlock()

	first := &mockBackend{name: "dup", audio: []byte("first")}
	second := &mockBackend{name: "dup", audio: []byte("second")}

	Register(first)
	Register(second) // should be ignored

	b, err := Get("dup")
	if err != nil {
		t.Fatal(err)
	}
	audio, _ := b.Speak("test", SpeakOpts{})
	if string(audio) != "first" {
		t.Errorf("expected first backend to be kept, got audio %q", audio)
	}
}

func TestElevenLabsHTTPClientTimeout(t *testing.T) {
	if elevenLabsHTTPClient == nil {
		t.Fatal("elevenLabsHTTPClient is nil")
	}
	if elevenLabsHTTPClient.Timeout == 0 {
		t.Error("elevenLabsHTTPClient should have a non-zero timeout")
	}
	if elevenLabsHTTPClient.Timeout.Seconds() != 30 {
		t.Errorf("expected 30s timeout, got %v", elevenLabsHTTPClient.Timeout)
	}
}
