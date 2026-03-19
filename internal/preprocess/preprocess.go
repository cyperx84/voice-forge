package preprocess

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cyperx84/voice-forge/internal/config"
)

// Manifest tracks processed files and their segments.
type Manifest struct {
	ProcessedAt string         `json:"processed_at"`
	Files       []ProcessedFile `json:"files"`
}

// ProcessedFile records a source file and its segments.
type ProcessedFile struct {
	Source   string   `json:"source"`
	Segments []string `json:"segments"`
}

// Run preprocesses audio files from inputDir into outputDir.
func Run(inputDir, outputDir string, force bool, cfg config.PreprocessConfig) (*Manifest, error) {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}
	segDir := filepath.Join(outputDir, "segments")
	if err := os.MkdirAll(segDir, 0755); err != nil {
		return nil, fmt.Errorf("create segments dir: %w", err)
	}

	files, err := findAudioFiles(inputDir)
	if err != nil {
		return nil, fmt.Errorf("find audio files: %w", err)
	}

	manifest := &Manifest{
		ProcessedAt: time.Now().Format(time.RFC3339),
	}

	for _, src := range files {
		if !force && isAlreadyProcessed(src, outputDir) {
			continue
		}

		segments, err := processFile(src, outputDir, segDir, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", filepath.Base(src), err)
			continue
		}

		manifest.Files = append(manifest.Files, ProcessedFile{
			Source:   src,
			Segments: segments,
		})
	}

	manifestPath := filepath.Join(outputDir, "manifest.json")
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal manifest: %w", err)
	}
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return nil, fmt.Errorf("write manifest: %w", err)
	}

	return manifest, nil
}

func findAudioFiles(dir string) ([]string, error) {
	var files []string
	exts := []string{".wav", ".mp3", ".ogg", ".flac", ".m4a", ".opus"}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		for _, valid := range exts {
			if ext == valid {
				files = append(files, filepath.Join(dir, e.Name()))
				break
			}
		}
	}
	return files, nil
}

func isAlreadyProcessed(src, outputDir string) bool {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return false
	}
	base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	normalized := filepath.Join(outputDir, base+".wav")
	outInfo, err := os.Stat(normalized)
	if err != nil {
		return false
	}
	return outInfo.ModTime().After(srcInfo.ModTime())
}

func processFile(src, outputDir, segDir string, cfg config.PreprocessConfig) ([]string, error) {
	base := strings.TrimSuffix(filepath.Base(src), filepath.Ext(src))
	normalized := filepath.Join(outputDir, base+".wav")

	// Step 1: Format normalize
	normArgs := NormalizeArgs(src, normalized, cfg)
	cmd := exec.Command("ffmpeg", normArgs...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("normalize: %w", err)
	}

	// Step 2: Denoise
	if cfg.Denoise {
		denoised := filepath.Join(outputDir, base+"_denoised.wav")
		denoiseArgs := DenoiseArgs(normalized, denoised)
		cmd = exec.Command("ffmpeg", denoiseArgs...)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("denoise: %w", err)
		}
		os.Rename(denoised, normalized)
	}

	// Step 3: VAD segmentation
	segments, err := segment(normalized, segDir, base, cfg)
	if err != nil {
		return nil, fmt.Errorf("segment: %w", err)
	}

	return segments, nil
}

// NormalizeArgs returns the ffmpeg arguments for format normalization.
func NormalizeArgs(input, output string, cfg config.PreprocessConfig) []string {
	return []string{
		"-y", "-i", input,
		"-ar", strconv.Itoa(cfg.SampleRate),
		"-ac", strconv.Itoa(cfg.Channels),
		"-sample_fmt", fmt.Sprintf("s%d", cfg.BitDepth),
		output,
	}
}

// DenoiseArgs returns the ffmpeg arguments for denoising.
func DenoiseArgs(input, output string) []string {
	return []string{
		"-y", "-i", input,
		"-af", "afftdn=nf=-20",
		output,
	}
}

func segment(wavPath, segDir, base string, cfg config.PreprocessConfig) ([]string, error) {
	// Use silencedetect to find silence boundaries
	args := []string{
		"-i", wavPath,
		"-af", "silencedetect=noise=-30dB:d=0.5",
		"-f", "null", "-",
	}
	cmd := exec.Command("ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("silencedetect: %w", err)
	}

	boundaries := ParseSilenceDetect(string(out))
	segments := SplitSegments(boundaries, cfg.MinSegment, cfg.MaxSegment)

	var result []string
	for i, seg := range segments {
		outPath := filepath.Join(segDir, fmt.Sprintf("%s_%03d.wav", base, i))
		cutArgs := []string{
			"-y", "-i", wavPath,
			"-ss", fmt.Sprintf("%.3f", seg.Start),
			"-t", fmt.Sprintf("%.3f", seg.End-seg.Start),
			"-c", "copy", outPath,
		}
		cmd := exec.Command("ffmpeg", cutArgs...)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			continue
		}
		result = append(result, outPath)
	}

	return result, nil
}

// Segment represents a time range.
type Segment struct {
	Start float64
	End   float64
}

// ParseSilenceDetect extracts silence_end timestamps from ffmpeg output.
func ParseSilenceDetect(output string) []float64 {
	re := regexp.MustCompile(`silence_end: ([\d.]+)`)
	matches := re.FindAllStringSubmatch(output, -1)
	var boundaries []float64
	for _, m := range matches {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			boundaries = append(boundaries, v)
		}
	}
	return boundaries
}

// SplitSegments creates segments from silence boundaries, filtering by duration.
func SplitSegments(boundaries []float64, minDur, maxDur float64) []Segment {
	if len(boundaries) == 0 {
		return nil
	}

	var segments []Segment
	start := 0.0
	for _, boundary := range boundaries {
		dur := boundary - start
		if dur >= minDur && dur <= maxDur {
			segments = append(segments, Segment{Start: start, End: boundary})
		}
		start = boundary
	}
	return segments
}
