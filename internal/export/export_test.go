package export

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cyperx84/voice-forge/internal/scoring"
)

func TestFormatLJSpeechRow(t *testing.T) {
	row := FormatLJSpeechRow("sample_001", "Hello world this is a test")
	expected := "sample_001|Hello world this is a test"
	if row != expected {
		t.Errorf("got %q, want %q", row, expected)
	}
}

func TestFilterByTier(t *testing.T) {
	files := []scoring.FileScore{
		{Path: "a.wav", Tier: scoring.TierGold},
		{Path: "b.wav", Tier: scoring.TierSilver},
		{Path: "c.wav", Tier: scoring.TierBronze},
		{Path: "d.wav", Tier: scoring.TierReject},
	}

	tests := []struct {
		threshold scoring.Tier
		wantCount int
	}{
		{scoring.TierGold, 1},
		{scoring.TierSilver, 2},
		{scoring.TierBronze, 3},
		{scoring.TierReject, 4},
	}

	for _, tt := range tests {
		t.Run(string(tt.threshold), func(t *testing.T) {
			filtered := FilterByTier(files, tt.threshold)
			if len(filtered) != tt.wantCount {
				t.Errorf("FilterByTier(%s) = %d files, want %d", tt.threshold, len(filtered), tt.wantCount)
			}
		})
	}
}

func TestExportLJSpeech(t *testing.T) {
	tmpDir := t.TempDir()

	// Create fake WAV files and transcripts
	audioDir := filepath.Join(tmpDir, "audio")
	os.MkdirAll(audioDir, 0755)

	wavPath := filepath.Join(audioDir, "test_001.wav")
	os.WriteFile(wavPath, []byte("fake wav data"), 0644)

	txtPath := filepath.Join(audioDir, "test_001.txt")
	os.WriteFile(txtPath, []byte("Hello this is a test transcript"), 0644)

	report := &scoring.Report{
		Files: []scoring.FileScore{
			{Path: wavPath, Tier: scoring.TierGold, Metrics: scoring.Metrics{SNR: 35, Duration: 5}},
		},
		Gold: 1,
	}

	outputDir := filepath.Join(tmpDir, "export")
	count, err := ExportLJSpeech(report, audioDir, outputDir, scoring.TierSilver)
	if err != nil {
		t.Fatalf("ExportLJSpeech() error: %v", err)
	}

	if count != 1 {
		t.Errorf("exported %d files, want 1", count)
	}

	// Check metadata.csv
	metadata, err := os.ReadFile(filepath.Join(outputDir, "metadata.csv"))
	if err != nil {
		t.Fatalf("read metadata.csv: %v", err)
	}

	line := strings.TrimSpace(string(metadata))
	if !strings.Contains(line, "test_001|Hello this is a test transcript") {
		t.Errorf("unexpected metadata: %q", line)
	}

	// Check WAV was copied
	if _, err := os.Stat(filepath.Join(outputDir, "wavs", "test_001.wav")); err != nil {
		t.Error("WAV file not copied to wavs/")
	}
}

func TestExportLJSpeechTierFiltering(t *testing.T) {
	tmpDir := t.TempDir()

	// Create audio files with transcripts
	for _, name := range []string{"gold", "silver", "bronze"} {
		wavPath := filepath.Join(tmpDir, name+".wav")
		os.WriteFile(wavPath, []byte("wav"), 0644)
		txtPath := filepath.Join(tmpDir, name+".txt")
		os.WriteFile(txtPath, []byte("transcript for "+name), 0644)
	}

	report := &scoring.Report{
		Files: []scoring.FileScore{
			{Path: filepath.Join(tmpDir, "gold.wav"), Tier: scoring.TierGold},
			{Path: filepath.Join(tmpDir, "silver.wav"), Tier: scoring.TierSilver},
			{Path: filepath.Join(tmpDir, "bronze.wav"), Tier: scoring.TierBronze},
		},
	}

	outputDir := filepath.Join(tmpDir, "export")
	count, err := ExportLJSpeech(report, tmpDir, outputDir, scoring.TierGold)
	if err != nil {
		t.Fatalf("ExportLJSpeech() error: %v", err)
	}

	if count != 1 {
		t.Errorf("exported %d files with gold threshold, want 1", count)
	}
}

func TestExportJSONL(t *testing.T) {
	tmpDir := t.TempDir()

	wavPath := filepath.Join(tmpDir, "sample.wav")
	os.WriteFile(wavPath, []byte("wav"), 0644)
	txtPath := filepath.Join(tmpDir, "sample.txt")
	os.WriteFile(txtPath, []byte("sample transcript"), 0644)

	report := &scoring.Report{
		Files: []scoring.FileScore{
			{Path: wavPath, Tier: scoring.TierSilver, Metrics: scoring.Metrics{SNR: 25, Duration: 7}},
		},
	}

	outputPath := filepath.Join(tmpDir, "export.jsonl")
	count, err := ExportJSONL(report, tmpDir, outputPath, scoring.TierBronze)
	if err != nil {
		t.Fatalf("ExportJSONL() error: %v", err)
	}

	if count != 1 {
		t.Errorf("exported %d files, want 1", count)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read jsonl: %v", err)
	}

	var entry JSONLEntry
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(data))), &entry); err != nil {
		t.Fatalf("unmarshal jsonl entry: %v", err)
	}

	if entry.Tier != "silver" {
		t.Errorf("tier = %q, want silver", entry.Tier)
	}
	if entry.Transcript != "sample transcript" {
		t.Errorf("transcript = %q, want 'sample transcript'", entry.Transcript)
	}
}
