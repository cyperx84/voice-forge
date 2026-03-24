package cmd

import "testing"

func TestSplitLines(t *testing.T) {
	got := splitLines("a\nb\n")
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("unexpected splitLines result: %#v", got)
	}
}

func TestParseVoiceBackend(t *testing.T) {
	meta := "name = \"cyperx\"\nbackend = \"chatterbox\"\nsamples = 5\n"
	if got := parseVoiceBackend(meta); got != "chatterbox" {
		t.Fatalf("parseVoiceBackend = %q, want chatterbox", got)
	}
}

func TestParseVoiceBackendMissing(t *testing.T) {
	if got := parseVoiceBackend("name = \"x\""); got != "unknown" {
		t.Fatalf("parseVoiceBackend = %q, want unknown", got)
	}
}
