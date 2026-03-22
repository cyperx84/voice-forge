package tts

import (
	"sync/atomic"
	"testing"
)

func TestSpeakScript_Parallel(t *testing.T) {
	ClearRegistry()

	var callCount atomic.Int32
	mock := &mockBackend{
		name:      "test",
		available: true,
		audio:     []byte("fake audio"),
	}

	// Override Speak to count calls
	Register(mock)

	lines := []string{"line one", "line two", "line three", "line four", "line five"}
	opts := SpeakOpts{Voice: "test"}
	scriptOpts := ScriptOpts{
		Workers: 2,
		OnProgress: func(done, total int, line string) {
			callCount.Add(1)
		},
	}

	segments, err := SpeakScript(lines, mock, opts, scriptOpts)
	if err != nil {
		t.Fatalf("SpeakScript error: %v", err)
	}

	if len(segments) != 5 {
		t.Fatalf("expected 5 segments, got %d", len(segments))
	}

	if int(callCount.Load()) != 5 {
		t.Fatalf("expected 5 progress callbacks, got %d", callCount.Load())
	}

	for i, seg := range segments {
		if string(seg) != "fake audio" {
			t.Fatalf("segment %d = %q, want %q", i, seg, "fake audio")
		}
	}
}

func TestSpeakScript_EmptyLines(t *testing.T) {
	mock := &mockBackend{name: "test", available: true, audio: []byte("x")}
	_, err := SpeakScript(nil, mock, SpeakOpts{}, ScriptOpts{})
	if err == nil {
		t.Fatal("expected error for empty lines")
	}
}
