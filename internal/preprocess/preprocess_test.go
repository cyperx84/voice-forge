package preprocess

import (
	"testing"

	"github.com/cyperx84/voice-forge/internal/config"
)

func TestNormalizeArgs(t *testing.T) {
	cfg := config.PreprocessConfig{
		SampleRate: 24000,
		Channels:   1,
		BitDepth:   16,
	}

	args := NormalizeArgs("/tmp/in.wav", "/tmp/out.wav", cfg)
	expected := []string{"-y", "-i", "/tmp/in.wav", "-ar", "24000", "-ac", "1", "-sample_fmt", "s16", "/tmp/out.wav"}

	if len(args) != len(expected) {
		t.Fatalf("expected %d args, got %d: %v", len(expected), len(args), args)
	}
	for i, a := range args {
		if a != expected[i] {
			t.Errorf("arg[%d] = %q, want %q", i, a, expected[i])
		}
	}
}

func TestDenoiseArgs(t *testing.T) {
	args := DenoiseArgs("/tmp/in.wav", "/tmp/out.wav")
	// -y -i input -af filter output = 6 args
	if len(args) != 6 {
		t.Fatalf("expected 6 args, got %d: %v", len(args), args)
	}
	if args[4] != "afftdn=nf=-20" {
		t.Errorf("expected afftdn filter, got %q", args[4])
	}
}

func TestParseSilenceDetect(t *testing.T) {
	output := `[silencedetect @ 0x7f8] silence_end: 1.234 | silence_duration: 0.6
[silencedetect @ 0x7f8] silence_end: 5.678 | silence_duration: 0.8
[silencedetect @ 0x7f8] silence_end: 10.000 | silence_duration: 1.2`

	boundaries := ParseSilenceDetect(output)
	if len(boundaries) != 3 {
		t.Fatalf("expected 3 boundaries, got %d", len(boundaries))
	}
	if boundaries[0] != 1.234 {
		t.Errorf("boundary[0] = %f, want 1.234", boundaries[0])
	}
	if boundaries[1] != 5.678 {
		t.Errorf("boundary[1] = %f, want 5.678", boundaries[1])
	}
	if boundaries[2] != 10.0 {
		t.Errorf("boundary[2] = %f, want 10.0", boundaries[2])
	}
}

func TestParseSilenceDetectEmpty(t *testing.T) {
	boundaries := ParseSilenceDetect("no silence detected")
	if len(boundaries) != 0 {
		t.Errorf("expected 0 boundaries, got %d", len(boundaries))
	}
}

func TestSplitSegments(t *testing.T) {
	tests := []struct {
		name       string
		boundaries []float64
		minDur     float64
		maxDur     float64
		wantCount  int
	}{
		{
			name:       "normal segments",
			boundaries: []float64{5.0, 10.0, 18.0},
			minDur:     3.0,
			maxDur:     15.0,
			wantCount:  3, // 0-5 (5s), 5-10 (5s), 10-18 (8s) all within range
		},
		{
			name:       "too short",
			boundaries: []float64{1.0, 2.0},
			minDur:     3.0,
			maxDur:     15.0,
			wantCount:  0,
		},
		{
			name:       "empty boundaries",
			boundaries: []float64{},
			minDur:     3.0,
			maxDur:     15.0,
			wantCount:  0,
		},
		{
			name:       "too long segment",
			boundaries: []float64{20.0},
			minDur:     3.0,
			maxDur:     15.0,
			wantCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			segments := SplitSegments(tt.boundaries, tt.minDur, tt.maxDur)
			if len(segments) != tt.wantCount {
				t.Errorf("got %d segments, want %d", len(segments), tt.wantCount)
			}
		})
	}
}

func TestSkipAlreadyProcessed(t *testing.T) {
	// isAlreadyProcessed returns false when output doesn't exist
	result := isAlreadyProcessed("/nonexistent/input.wav", "/nonexistent/output")
	if result {
		t.Error("should return false for nonexistent files")
	}
}
