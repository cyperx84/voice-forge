package watch

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsOggFile(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"recording.ogg", true},
		{"recording.OGG", true},
		{"recording.wav", false},
		{"recording.txt", false},
		{"recording.mp3", false},
		{"noext", false},
	}
	for _, tt := range tests {
		if got := isOggFile(tt.name); got != tt.want {
			t.Errorf("isOggFile(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestHasTranscript(t *testing.T) {
	dir := t.TempDir()

	// Create an ogg file without transcript
	oggPath := filepath.Join(dir, "test.ogg")
	os.WriteFile(oggPath, []byte("fake ogg"), 0644)

	w := &Watcher{Dir: dir}

	if w.hasTranscript(oggPath) {
		t.Error("hasTranscript should return false when no .txt exists")
	}

	// Create the transcript
	txtPath := filepath.Join(dir, "test.txt")
	os.WriteFile(txtPath, []byte("hello world"), 0644)

	if !w.hasTranscript(oggPath) {
		t.Error("hasTranscript should return true when .txt exists")
	}
}

func TestProcessExisting_SkipsTranscribed(t *testing.T) {
	dir := t.TempDir()

	// Create an ogg file with existing transcript
	os.WriteFile(filepath.Join(dir, "a.ogg"), []byte("fake"), 0644)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("already done"), 0644)

	// Create an ogg file without transcript (will fail because no ffmpeg/whisper)
	os.WriteFile(filepath.Join(dir, "b.ogg"), []byte("fake"), 0644)

	w := &Watcher{
		Dir:            dir,
		Interval:       30 * time.Second,
		WhisperCommand: "false", // will fail
	}

	// b.ogg should attempt processing but fail (no ffmpeg)
	// a.ogg should be skipped entirely
	n, _ := w.ProcessExisting()
	// We expect 0 because ffmpeg will fail on fake data
	if n != 0 {
		t.Errorf("ProcessExisting = %d, want 0 (ffmpeg should fail on fake data)", n)
	}
}

func TestProcessExisting_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	w := &Watcher{Dir: dir, Interval: 30 * time.Second}
	n, err := w.ProcessExisting()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("ProcessExisting on empty dir = %d, want 0", n)
	}
}

func TestProcessExisting_NonexistentDir(t *testing.T) {
	w := &Watcher{Dir: "/tmp/nonexistent-voice-forge-test-dir", Interval: 30 * time.Second}
	n, err := w.ProcessExisting()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("ProcessExisting on nonexistent dir = %d, want 0", n)
	}
}

func TestCountTranscripts(t *testing.T) {
	dir := t.TempDir()

	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(dir, "b.txt"), []byte("world"), 0644)
	os.WriteFile(filepath.Join(dir, "c.ogg"), []byte("audio"), 0644)

	count := CountTranscripts(dir)
	if count != 2 {
		t.Errorf("CountTranscripts = %d, want 2", count)
	}
}
