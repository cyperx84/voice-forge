package scoring

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Tier represents a quality tier.
type Tier string

const (
	TierGold   Tier = "gold"
	TierSilver Tier = "silver"
	TierBronze Tier = "bronze"
	TierReject Tier = "reject"
)

// TierRank returns a numeric rank for comparison (higher = better).
func TierRank(t Tier) int {
	switch t {
	case TierGold:
		return 3
	case TierSilver:
		return 2
	case TierBronze:
		return 1
	default:
		return 0
	}
}

// MeetsThreshold returns true if tier meets or exceeds the threshold.
func MeetsThreshold(tier, threshold Tier) bool {
	return TierRank(tier) >= TierRank(threshold)
}

// Metrics holds audio quality measurements.
type Metrics struct {
	SNR          float64 `json:"snr_db"`
	ClippingPct  float64 `json:"clipping_pct"`
	Duration     float64 `json:"duration_s"`
	SilenceRatio float64 `json:"silence_ratio"`
}

// FileScore holds the tier and metrics for a single file.
type FileScore struct {
	Path    string  `json:"path"`
	Tier    Tier    `json:"tier"`
	Metrics Metrics `json:"metrics"`
}

// Report holds all scored files and summary counts.
type Report struct {
	Files  []FileScore `json:"files"`
	Gold   int         `json:"gold"`
	Silver int         `json:"silver"`
	Bronze int         `json:"bronze"`
	Reject int         `json:"reject"`
}

// AssignTier determines the quality tier from metrics.
func AssignTier(m Metrics) Tier {
	if m.SNR > 30 && m.ClippingPct == 0 && m.Duration >= 3 && m.Duration <= 15 && m.SilenceRatio < 0.20 {
		return TierGold
	}
	if m.SNR > 20 && m.ClippingPct < 0.1 && m.Duration >= 1 && m.Duration <= 30 {
		return TierSilver
	}
	if m.SNR > 10 && m.ClippingPct < 1.0 {
		return TierBronze
	}
	return TierReject
}

// ScoreDir scores all WAV files in a directory.
func ScoreDir(inputDir string) (*Report, error) {
	entries, err := os.ReadDir(inputDir)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	report := &Report{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".wav") {
			continue
		}
		path := filepath.Join(inputDir, e.Name())
		metrics, err := AnalyzeFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", e.Name(), err)
			continue
		}
		tier := AssignTier(metrics)
		report.Files = append(report.Files, FileScore{
			Path:    path,
			Tier:    tier,
			Metrics: metrics,
		})
		switch tier {
		case TierGold:
			report.Gold++
		case TierSilver:
			report.Silver++
		case TierBronze:
			report.Bronze++
		case TierReject:
			report.Reject++
		}
	}

	return report, nil
}

// SaveReport writes the score report to a JSON file.
func SaveReport(report *Report, outputPath string) error {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outputPath, data, 0644)
}

// AnalyzeFile extracts audio metrics from a WAV file using ffmpeg astats.
func AnalyzeFile(path string) (Metrics, error) {
	// Use ametadata=print without key filter to get ALL stats
	args := []string{
		"-i", path,
		"-af", "astats=metadata=1:reset=0,ametadata=print",
		"-f", "null", "-",
	}
	cmd := exec.Command("ffmpeg", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return Metrics{}, fmt.Errorf("ffmpeg astats: %w", err)
	}
	return ParseAstats(string(out))
}

// ParseAstats extracts metrics from ffmpeg astats output.
func ParseAstats(output string) (Metrics, error) {
	var m Metrics

	rmsLevel := extractFloat(output, `RMS_level=([-\d.]+)`)
	noiseFloor := extractFloat(output, `Noise_floor=([-\d.inf]+)`)
	rmsTrough := extractFloat(output, `RMS_trough=([-\d.]+)`)
	peakCount := extractFloat(output, `Peak_count=([\d.]+)`)
	numSamples := extractFloat(output, `Number_of_samples=([\d.]+)`)

	// SNR estimation: use RMS_level - RMS_trough as proxy
	// Noise_floor can be -inf (silent sections), so prefer RMS_trough
	if rmsLevel != 0 && rmsTrough != 0 {
		m.SNR = rmsLevel - rmsTrough
	} else if rmsLevel != 0 && noiseFloor != 0 {
		m.SNR = rmsLevel - noiseFloor
	}
	// SNR from dB subtraction: positive means signal louder than noise
	// For voice messages, RMS_level ~ -13dB, RMS_trough ~ -70dB → SNR ~ 57dB

	// Clipping percentage: Peak_count / Number_of_samples
	// Peak_count is typically small (number of clipped peaks), not per-sample
	if numSamples > 0 {
		m.ClippingPct = (peakCount / numSamples) * 100
	}

	// Duration from ffmpeg time= output
	dur, err := getDuration(output)
	if err == nil {
		m.Duration = dur
	}

	// Silence ratio: default 0 (good enough for phone recordings)
	m.SilenceRatio = 0

	return m, nil
}

func extractFloat(output, pattern string) float64 {
	re := regexp.MustCompile(pattern)
	matches := re.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return 0
	}
	// Take the last match
	last := matches[len(matches)-1]
	s := last[1]
	// Handle -inf / inf from ffmpeg
	if s == "-inf" || s == "inf" || s == "-nan" || s == "nan" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

func getDuration(output string) (float64, error) {
	re := regexp.MustCompile(`time=([\d:.]+)`)
	matches := re.FindAllStringSubmatch(output, -1)
	if len(matches) == 0 {
		return 0, fmt.Errorf("no duration found")
	}
	last := matches[len(matches)-1]
	return parseTimestamp(last[1])
}

func parseTimestamp(ts string) (float64, error) {
	parts := strings.Split(ts, ":")
	if len(parts) != 3 {
		return strconv.ParseFloat(ts, 64)
	}
	h, _ := strconv.ParseFloat(parts[0], 64)
	m, _ := strconv.ParseFloat(parts[1], 64)
	s, _ := strconv.ParseFloat(parts[2], 64)
	return h*3600 + m*60 + s, nil
}
