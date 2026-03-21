package audioout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscordFFmpegArgs(t *testing.T) {
	args := DiscordFFmpegArgs("in.wav", "out.mp3")

	got := strings.Join(args, " ")
	wantParts := []string{
		"-i in.wav",
		"-ar 48000",
		"-ac 1",
		"-codec:a libmp3lame",
		"-b:a 128k",
		"out.mp3",
	}
	for _, want := range wantParts {
		if !strings.Contains(got, want) {
			t.Fatalf("expected ffmpeg args to contain %q, got %q", want, got)
		}
	}
}

func TestListenPagePath(t *testing.T) {
	got := ListenPagePath("/tmp/clip.mp3")
	if got != "/tmp/clip.listen.html" {
		t.Fatalf("ListenPagePath = %q, want %q", got, "/tmp/clip.listen.html")
	}
}

func TestWriteListenPage(t *testing.T) {
	dir := t.TempDir()
	audioPath := filepath.Join(dir, "clip.mp3")
	if err := os.WriteFile(audioPath, []byte("fake mp3 bytes"), 0644); err != nil {
		t.Fatal(err)
	}

	pagePath, err := WriteListenPage(audioPath, "Test <Clip>", "Line one\nLine two")
	if err != nil {
		t.Fatalf("WriteListenPage error: %v", err)
	}

	html, err := os.ReadFile(pagePath)
	if err != nil {
		t.Fatal(err)
	}
	s := string(html)

	if !strings.Contains(s, "<title>Test &lt;Clip&gt;</title>") {
		t.Fatalf("listen page should escape title, got %q", s)
	}
	if !strings.Contains(s, "audio/mpeg") {
		t.Fatalf("listen page should include audio MIME, got %q", s)
	}
	if !strings.Contains(s, "Line one\nLine two") {
		t.Fatalf("listen page should include text, got %q", s)
	}
	if !strings.Contains(s, "ZmFrZSBtcDMgYnl0ZXM=") {
		t.Fatalf("listen page should embed base64 audio, got %q", s)
	}
}
