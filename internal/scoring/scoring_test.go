package scoring

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAssignTier(t *testing.T) {
	tests := []struct {
		name    string
		metrics Metrics
		want    Tier
	}{
		{
			name: "gold - perfect recording",
			metrics: Metrics{
				SNR:          35.0,
				ClippingPct:  0.0,
				Duration:     8.0,
				SilenceRatio: 0.10,
			},
			want: TierGold,
		},
		{
			name: "silver - decent recording",
			metrics: Metrics{
				SNR:          25.0,
				ClippingPct:  0.05,
				Duration:     20.0,
				SilenceRatio: 0.30,
			},
			want: TierSilver,
		},
		{
			name: "bronze - mediocre recording",
			metrics: Metrics{
				SNR:          15.0,
				ClippingPct:  0.5,
				Duration:     45.0,
				SilenceRatio: 0.50,
			},
			want: TierBronze,
		},
		{
			name: "reject - bad recording",
			metrics: Metrics{
				SNR:          5.0,
				ClippingPct:  2.0,
				Duration:     60.0,
				SilenceRatio: 0.80,
			},
			want: TierReject,
		},
		{
			name: "gold boundary - exactly at thresholds",
			metrics: Metrics{
				SNR:          30.1,
				ClippingPct:  0.0,
				Duration:     3.0,
				SilenceRatio: 0.19,
			},
			want: TierGold,
		},
		{
			name: "silver - gold SNR but too long",
			metrics: Metrics{
				SNR:          35.0,
				ClippingPct:  0.0,
				Duration:     20.0,
				SilenceRatio: 0.10,
			},
			want: TierSilver,
		},
		{
			name: "silver - gold SNR but too much silence",
			metrics: Metrics{
				SNR:          35.0,
				ClippingPct:  0.0,
				Duration:     8.0,
				SilenceRatio: 0.25,
			},
			want: TierSilver,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AssignTier(tt.metrics)
			if got != tt.want {
				t.Errorf("AssignTier() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestMeetsThreshold(t *testing.T) {
	tests := []struct {
		tier      Tier
		threshold Tier
		want      bool
	}{
		{TierGold, TierGold, true},
		{TierGold, TierSilver, true},
		{TierGold, TierBronze, true},
		{TierSilver, TierGold, false},
		{TierSilver, TierSilver, true},
		{TierBronze, TierSilver, false},
		{TierReject, TierBronze, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.tier)+"_vs_"+string(tt.threshold), func(t *testing.T) {
			got := MeetsThreshold(tt.tier, tt.threshold)
			if got != tt.want {
				t.Errorf("MeetsThreshold(%s, %s) = %v, want %v", tt.tier, tt.threshold, got, tt.want)
			}
		})
	}
}

func TestTierRank(t *testing.T) {
	if TierRank(TierGold) <= TierRank(TierSilver) {
		t.Error("gold should rank higher than silver")
	}
	if TierRank(TierSilver) <= TierRank(TierBronze) {
		t.Error("silver should rank higher than bronze")
	}
	if TierRank(TierBronze) <= TierRank(TierReject) {
		t.Error("bronze should rank higher than reject")
	}
}

func TestParseAstats(t *testing.T) {
	output := `[Parsed_ametadata_1 @ 0x1234] lavfi.astats.Overall.RMS_level=-20.5
[Parsed_ametadata_1 @ 0x1234] lavfi.astats.Overall.Noise_floor=-55.3
[Parsed_ametadata_1 @ 0x1234] lavfi.astats.Overall.Peak_count=5
[Parsed_ametadata_1 @ 0x1234] lavfi.astats.Overall.Number_of_samples=48000
size=N/A time=00:00:05.00 bitrate=N/A`

	m, err := ParseAstats(output)
	if err != nil {
		t.Fatalf("ParseAstats() error: %v", err)
	}

	// SNR = RMS - noise floor = -20.5 - (-55.3) = 34.8
	expectedSNR := 34.8
	if m.SNR < expectedSNR-0.1 || m.SNR > expectedSNR+0.1 {
		t.Errorf("SNR = %f, want ~%f", m.SNR, expectedSNR)
	}

	// Clipping = 5/48000 * 100 ~= 0.0104%
	if m.ClippingPct < 0.01 || m.ClippingPct > 0.02 {
		t.Errorf("ClippingPct = %f, want ~0.0104", m.ClippingPct)
	}

	// Duration from time= field
	if m.Duration != 5.0 {
		t.Errorf("Duration = %f, want 5.0", m.Duration)
	}
}

func TestSaveReport(t *testing.T) {
	tmpDir := t.TempDir()
	report := &Report{
		Files: []FileScore{
			{Path: "/tmp/a.wav", Tier: TierGold, Metrics: Metrics{SNR: 35}},
			{Path: "/tmp/b.wav", Tier: TierSilver, Metrics: Metrics{SNR: 25}},
		},
		Gold:   1,
		Silver: 1,
	}

	outPath := filepath.Join(tmpDir, "scores.json")
	if err := SaveReport(report, outPath); err != nil {
		t.Fatalf("SaveReport() error: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read scores: %v", err)
	}

	var loaded Report
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(loaded.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(loaded.Files))
	}
	if loaded.Gold != 1 {
		t.Errorf("expected 1 gold, got %d", loaded.Gold)
	}
}
