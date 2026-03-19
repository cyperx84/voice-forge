package export

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cyperx84/voice-forge/internal/scoring"
)

// LJSpeechEntry represents a single LJSpeech metadata row.
type LJSpeechEntry struct {
	Filename   string
	Transcript string
}

// JSONLEntry represents a single JSONL export row.
type JSONLEntry struct {
	Path       string  `json:"path"`
	Transcript string  `json:"transcript"`
	Duration   float64 `json:"duration"`
	Tier       string  `json:"tier"`
	SNR        float64 `json:"snr_db"`
}

// ExportLJSpeech exports scored files in LJSpeech format.
func ExportLJSpeech(report *scoring.Report, transcriptDir, outputDir string, threshold scoring.Tier) (int, error) {
	wavsDir := filepath.Join(outputDir, "wavs")
	if err := os.MkdirAll(wavsDir, 0755); err != nil {
		return 0, fmt.Errorf("create wavs dir: %w", err)
	}

	metadataPath := filepath.Join(outputDir, "metadata.csv")
	f, err := os.Create(metadataPath)
	if err != nil {
		return 0, fmt.Errorf("create metadata.csv: %w", err)
	}
	defer f.Close()

	count := 0
	for _, fs := range report.Files {
		if !scoring.MeetsThreshold(fs.Tier, threshold) {
			continue
		}

		transcript := loadTranscript(fs.Path, transcriptDir)
		if transcript == "" {
			continue
		}

		base := strings.TrimSuffix(filepath.Base(fs.Path), filepath.Ext(fs.Path))

		// Copy WAV to wavs/
		destPath := filepath.Join(wavsDir, base+".wav")
		if err := copyFile(fs.Path, destPath); err != nil {
			fmt.Fprintf(os.Stderr, "warning: copy %s: %v\n", base, err)
			continue
		}

		// Write metadata row: filename|transcript
		fmt.Fprintf(f, "%s|%s\n", base, transcript)
		count++
	}

	return count, nil
}

// ExportJSONL exports scored files in JSONL format.
func ExportJSONL(report *scoring.Report, transcriptDir, outputPath string, threshold scoring.Tier) (int, error) {
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return 0, fmt.Errorf("create output dir: %w", err)
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return 0, fmt.Errorf("create output: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	count := 0
	for _, fs := range report.Files {
		if !scoring.MeetsThreshold(fs.Tier, threshold) {
			continue
		}

		transcript := loadTranscript(fs.Path, transcriptDir)
		entry := JSONLEntry{
			Path:       fs.Path,
			Transcript: transcript,
			Duration:   fs.Metrics.Duration,
			Tier:       string(fs.Tier),
			SNR:        fs.Metrics.SNR,
		}
		if err := enc.Encode(entry); err != nil {
			continue
		}
		count++
	}

	return count, nil
}

// FilterByTier returns only files that meet the threshold.
func FilterByTier(files []scoring.FileScore, threshold scoring.Tier) []scoring.FileScore {
	var result []scoring.FileScore
	for _, f := range files {
		if scoring.MeetsThreshold(f.Tier, threshold) {
			result = append(result, f)
		}
	}
	return result
}

// FormatLJSpeechRow formats a single LJSpeech metadata row.
func FormatLJSpeechRow(filename, transcript string) string {
	return fmt.Sprintf("%s|%s", filename, transcript)
}

func loadTranscript(audioPath, transcriptDir string) string {
	base := strings.TrimSuffix(filepath.Base(audioPath), filepath.Ext(audioPath))

	// Try transcript file alongside audio
	txtPath := strings.TrimSuffix(audioPath, filepath.Ext(audioPath)) + ".txt"
	if data, err := os.ReadFile(txtPath); err == nil {
		return strings.TrimSpace(string(data))
	}

	// Try in transcriptDir
	if transcriptDir != "" {
		txtPath = filepath.Join(transcriptDir, base+".txt")
		if data, err := os.ReadFile(txtPath); err == nil {
			return strings.TrimSpace(string(data))
		}
	}

	return ""
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
