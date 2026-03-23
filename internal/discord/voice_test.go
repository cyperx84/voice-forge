package discord

import (
	"encoding/base64"
	"testing"
)

func TestParseRMSLevels(t *testing.T) {
	output := `lavfi.astats.Overall.RMS_level=-20.5
lavfi.astats.Overall.RMS_level=-15.3
lavfi.astats.Overall.RMS_level=-25.0
`
	values := parseRMSLevels(output)
	if len(values) != 3 {
		t.Fatalf("expected 3 values, got %d", len(values))
	}
	if values[0] != -20.5 {
		t.Fatalf("values[0] = %f, want -20.5", values[0])
	}
}

func TestDownsample(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	result := downsample(values, 5)
	if len(result) != 5 {
		t.Fatalf("expected 5 values, got %d", len(result))
	}
	// Each bucket should average 2 consecutive values
	if result[0] != 1.5 {
		t.Fatalf("result[0] = %f, want 1.5", result[0])
	}
}

func TestDownsample_NoOp(t *testing.T) {
	values := []float64{1, 2, 3}
	result := downsample(values, 10)
	if len(result) != 3 {
		t.Fatalf("expected 3 values (unchanged), got %d", len(result))
	}
}

func TestNormalizeAmplitudes(t *testing.T) {
	// -60 dB should map to 0, 0 dB should map to 255
	dbValues := []float64{-60, -30, 0}
	result := normalizeAmplitudes(dbValues)
	if len(result) != 3 {
		t.Fatalf("expected 3 bytes, got %d", len(result))
	}
	if result[0] != 0 {
		t.Fatalf("result[0] = %d, want 0 (silence)", result[0])
	}
	if result[1] != 127 { // -30 is midpoint
		t.Fatalf("result[1] = %d, want ~127 (midpoint)", result[1])
	}
	if result[2] != 255 {
		t.Fatalf("result[2] = %d, want 255 (max)", result[2])
	}
}

func TestVoiceMetaPath(t *testing.T) {
	got := VoiceMetaPath("/tmp/clip.mp3")
	if got != "/tmp/clip.voice.json" {
		t.Fatalf("VoiceMetaPath = %q, want %q", got, "/tmp/clip.voice.json")
	}
}

func TestNormalizeAmplitudes_ClampsBeyondRange(t *testing.T) {
	dbValues := []float64{-100, 10}
	result := normalizeAmplitudes(dbValues)
	if result[0] != 0 {
		t.Fatalf("result[0] = %d, want 0 (clamped)", result[0])
	}
	if result[1] != 255 {
		t.Fatalf("result[1] = %d, want 255 (clamped)", result[1])
	}
}

func TestBase64Waveform(t *testing.T) {
	// Ensure base64 encoding/decoding works for typical waveform data
	data := []byte{0, 64, 128, 192, 255}
	encoded := base64.StdEncoding.EncodeToString(data)
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(decoded) != len(data) {
		t.Fatalf("decoded length = %d, want %d", len(decoded), len(data))
	}
}
