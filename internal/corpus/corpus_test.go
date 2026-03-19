package corpus

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTokenize(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"hello world", 2},
		{"it's a test", 3},
		{"one, two, three!", 3},
		{"", 0},
		{"don't stop believin'", 3},
	}

	for _, tt := range tests {
		got := tokenize(tt.input)
		if len(got) != tt.want {
			t.Errorf("tokenize(%q) returned %d words, want %d: %v", tt.input, len(got), tt.want, got)
		}
	}
}

func TestTokenizeLowercase(t *testing.T) {
	words := tokenize("Hello WORLD Test")
	for _, w := range words {
		if w != "hello" && w != "world" && w != "test" {
			t.Errorf("expected lowercase word, got %q", w)
		}
	}
}

func TestIsStopWord(t *testing.T) {
	stops := []string{"the", "a", "is", "i", "you", "don't", "i'm"}
	for _, w := range stops {
		if !isStopWord(w) {
			t.Errorf("expected %q to be a stop word", w)
		}
	}

	nonStops := []string{"hello", "code", "voice", "corpus"}
	for _, w := range nonStops {
		if isStopWord(w) {
			t.Errorf("expected %q to NOT be a stop word", w)
		}
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
	}{
		{"16s", 16 * time.Second},
		{"37s", 37 * time.Second},
		{"1m30s", 90 * time.Second},
		{"42", 42 * time.Second},
		{"", 0},
		{"invalid", 0},
	}

	for _, tt := range tests {
		got := parseDuration(tt.input)
		if got != tt.want {
			t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestParseManifest(t *testing.T) {
	tmpDir := t.TempDir()
	manifestPath := filepath.Join(tmpDir, "manifest.txt")

	content := `060d7a16-7917-4501-8b14-3c4b48bdd845|16s|
I want to test out a thread in ops
0ba31b78-ee57-4e36-ac74-c11d87785a2d|7s|
All right the voice should be working now
`
	if err := os.WriteFile(manifestPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	recs, err := ParseManifest(manifestPath)
	if err != nil {
		t.Fatalf("ParseManifest() error: %v", err)
	}

	if len(recs) != 2 {
		t.Fatalf("expected 2 recordings, got %d", len(recs))
	}

	if recs[0].UUID != "060d7a16-7917-4501-8b14-3c4b48bdd845" {
		t.Errorf("unexpected UUID: %s", recs[0].UUID)
	}
	if recs[0].Duration != 16*time.Second {
		t.Errorf("expected 16s duration, got %v", recs[0].Duration)
	}
	if recs[0].Transcript != "I want to test out a thread in ops" {
		t.Errorf("unexpected transcript: %q", recs[0].Transcript)
	}
	if recs[1].Duration != 7*time.Second {
		t.Errorf("expected 7s duration, got %v", recs[1].Duration)
	}
}

func TestParseManifestNonExistent(t *testing.T) {
	recs, err := ParseManifest("/nonexistent/manifest.txt")
	if err != nil {
		t.Errorf("expected nil error for nonexistent file, got %v", err)
	}
	if recs != nil {
		t.Errorf("expected nil recordings for nonexistent file")
	}
}

func TestReadTranscripts(t *testing.T) {
	tmpDir := t.TempDir()

	// Create UUID-named transcript files
	uuids := []string{
		"060d7a16-7917-4501-8b14-3c4b48bdd845",
		"0ba31b78-ee57-4e36-ac74-c11d87785a2d",
	}
	for i, uuid := range uuids {
		content := "transcript " + string(rune('A'+i))
		if err := os.WriteFile(filepath.Join(tmpDir, uuid+".txt"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Create a non-UUID txt file (should be ignored)
	if err := os.WriteFile(filepath.Join(tmpDir, "manifest.txt"), []byte("not a transcript"), 0644); err != nil {
		t.Fatal(err)
	}

	transcripts, err := ReadTranscripts([]string{tmpDir})
	if err != nil {
		t.Fatalf("ReadTranscripts() error: %v", err)
	}

	if len(transcripts) != 2 {
		t.Errorf("expected 2 transcripts, got %d", len(transcripts))
	}
}

func TestReadTranscriptsSubdir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create transcripts/ subdirectory
	transcriptsDir := filepath.Join(tmpDir, "transcripts")
	if err := os.MkdirAll(transcriptsDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(transcriptsDir, "2026-02-26_104500.txt"), []byte("test transcript"), 0644); err != nil {
		t.Fatal(err)
	}

	transcripts, err := ReadTranscripts([]string{tmpDir})
	if err != nil {
		t.Fatalf("ReadTranscripts() error: %v", err)
	}

	if len(transcripts) != 1 {
		t.Errorf("expected 1 transcript from subdirectory, got %d", len(transcripts))
	}
}

func TestComputeStats(t *testing.T) {
	recordings := []Recording{
		{UUID: "aaa", Duration: 10 * time.Second, Transcript: "hello world"},
		{UUID: "bbb", Duration: 20 * time.Second, Transcript: "testing one two three"},
	}
	transcripts := []string{"voice corpus test", "another test here"}

	stats := ComputeStats(recordings, transcripts)

	if stats.TotalRecordings != 2 {
		t.Errorf("expected 2 recordings, got %d", stats.TotalRecordings)
	}
	if stats.TotalDuration != 30*time.Second {
		t.Errorf("expected 30s total duration, got %v", stats.TotalDuration)
	}
	if stats.AvgDuration != 15*time.Second {
		t.Errorf("expected 15s avg duration, got %v", stats.AvgDuration)
	}
	if stats.TotalWords == 0 {
		t.Error("expected non-zero total words")
	}
	if stats.UniqueWords == 0 {
		t.Error("expected non-zero unique words")
	}
}

func TestComputeStatsTopWords(t *testing.T) {
	// "voice" repeated many times should appear in top words
	transcripts := []string{}
	for i := 0; i < 20; i++ {
		transcripts = append(transcripts, "voice corpus voice forge voice")
	}

	stats := ComputeStats(nil, transcripts)

	if len(stats.TopWords) == 0 {
		t.Fatal("expected top words")
	}
	if stats.TopWords[0].Word != "voice" {
		t.Errorf("expected 'voice' as top word, got %q", stats.TopWords[0].Word)
	}
}

func TestGetFileModTimes(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some files
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	earliest, latest := GetFileModTimes([]string{tmpDir})
	if earliest.IsZero() || latest.IsZero() {
		t.Error("expected non-zero times")
	}
	if latest.Before(earliest) {
		t.Error("latest should not be before earliest")
	}
}
