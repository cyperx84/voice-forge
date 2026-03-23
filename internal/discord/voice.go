package discord

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cyperx84/voice-forge/internal/ffmpeg"
)

// VoiceMeta holds metadata needed for Discord's native voice message player.
type VoiceMeta struct {
	DurationSecs float64 `json:"duration_secs"`
	Waveform     string  `json:"waveform"` // base64-encoded amplitude array (≤256 bytes)
}

// ComputeVoiceMeta extracts duration and generates a waveform from an audio file.
// The waveform is a base64-encoded array of amplitude values (0-255), sampled at
// regular intervals across the audio, suitable for Discord's voice message player.
func ComputeVoiceMeta(audioPath string, ffCfg ffmpeg.Config) (*VoiceMeta, error) {
	duration, err := probeDuration(audioPath, ffCfg)
	if err != nil {
		return nil, fmt.Errorf("getting duration: %w", err)
	}

	waveform, err := generateWaveform(audioPath, ffCfg)
	if err != nil {
		return nil, fmt.Errorf("generating waveform: %w", err)
	}

	return &VoiceMeta{
		DurationSecs: duration,
		Waveform:     waveform,
	}, nil
}

// SaveVoiceMeta writes the voice metadata as a JSON sidecar file.
func SaveVoiceMeta(meta *VoiceMeta, outputPath string) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling voice meta: %w", err)
	}
	return os.WriteFile(outputPath, data, 0644)
}

// VoiceMetaPath returns the sidecar metadata path for an audio file.
func VoiceMetaPath(audioPath string) string {
	return strings.TrimSuffix(audioPath, ".mp3") + ".voice.json"
}

// probeDuration uses ffprobe to get the audio duration in seconds.
func probeDuration(audioPath string, ffCfg ffmpeg.Config) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := ffmpeg.ProbeCommand(ctx, ffCfg,
		"-v", "quiet",
		"-show_entries", "format=duration",
		"-of", "default=noprint_wrappers=1:nokey=1",
		audioPath,
	)

	out, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("ffprobe: %w", err)
	}

	var dur float64
	if _, err := fmt.Sscanf(strings.TrimSpace(string(out)), "%f", &dur); err != nil {
		return 0, fmt.Errorf("parsing duration: %w", err)
	}
	return dur, nil
}

// generateWaveform creates a base64-encoded amplitude waveform (256 samples max).
// Uses ffmpeg astats to extract per-frame RMS levels, then downsamples to 256 values.
func generateWaveform(audioPath string, ffCfg ffmpeg.Config) (string, error) {
	// Extract per-frame RMS values using volumedetect
	out, err := ffmpeg.Run(ffCfg,
		"-i", audioPath,
		"-af", "astats=metadata=1:reset=1,ametadata=print:key=lavfi.astats.Overall.RMS_level",
		"-f", "null", "-",
	)
	if err != nil {
		// Fallback: return a flat waveform rather than fail completely
		flat := make([]byte, 64)
		for i := range flat {
			flat[i] = 128
		}
		return base64.StdEncoding.EncodeToString(flat), nil
	}

	// Parse RMS levels from output
	rmsValues := parseRMSLevels(string(out))
	if len(rmsValues) == 0 {
		flat := make([]byte, 64)
		for i := range flat {
			flat[i] = 128
		}
		return base64.StdEncoding.EncodeToString(flat), nil
	}

	// Downsample to 256 values max
	const maxSamples = 256
	samples := downsample(rmsValues, maxSamples)

	// Normalize to 0-255 range
	normalized := normalizeAmplitudes(samples)

	return base64.StdEncoding.EncodeToString(normalized), nil
}

// parseRMSLevels extracts RMS dB values from ffmpeg ametadata output.
func parseRMSLevels(output string) []float64 {
	re := regexp.MustCompile(`lavfi\.astats\.Overall\.RMS_level=([-\d.]+)`)
	matches := re.FindAllStringSubmatch(output, -1)
	var values []float64
	for _, m := range matches {
		if v, err := strconv.ParseFloat(m[1], 64); err == nil {
			values = append(values, v)
		}
	}
	return values
}

// downsample reduces a slice of float64 values to the target count by averaging.
func downsample(values []float64, target int) []float64 {
	if len(values) <= target {
		return values
	}
	result := make([]float64, target)
	step := float64(len(values)) / float64(target)
	for i := 0; i < target; i++ {
		start := int(float64(i) * step)
		end := int(float64(i+1) * step)
		if end > len(values) {
			end = len(values)
		}
		sum := 0.0
		for j := start; j < end; j++ {
			sum += values[j]
		}
		result[i] = sum / float64(end-start)
	}
	return result
}

// normalizeAmplitudes converts dB values to 0-255 byte range.
func normalizeAmplitudes(dbValues []float64) []byte {
	if len(dbValues) == 0 {
		return nil
	}

	// RMS levels are typically between -60 (silent) and 0 (max)
	result := make([]byte, len(dbValues))
	for i, db := range dbValues {
		// Clamp to -60..0 range
		if db < -60 {
			db = -60
		}
		if db > 0 {
			db = 0
		}
		// Map -60..0 to 0..255
		normalized := (db + 60) / 60 * 255
		result[i] = byte(math.Min(255, math.Max(0, normalized)))
	}
	return result
}
