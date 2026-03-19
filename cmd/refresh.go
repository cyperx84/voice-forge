package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/cyperx84/voice-forge/internal/analyzer"
	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/corpus"
	"github.com/spf13/cobra"
)

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Re-run style analysis if corpus has grown enough",
	Long: `Smart re-analysis of the voice corpus.

Skips if the profile is less than 24h old and no significant new data.
Re-analyzes if the corpus has grown by 10%+ or 20+ new transcripts.
Use --force to always re-analyze.

Designed to be called by OpenClaw cron daily.

Examples:
  forge refresh
  forge refresh --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		force, _ := cmd.Flags().GetBool("force")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		paths := cfg.CorpusPaths()
		transcripts, err := corpus.ReadTranscripts(paths)
		if err != nil {
			return fmt.Errorf("reading transcripts: %w", err)
		}

		if len(transcripts) == 0 {
			fmt.Println("No transcripts found in corpus directories.")
			return nil
		}

		profileDir := cfg.ProfileDir()
		stylePath := filepath.Join(profileDir, "style.json")

		if !force {
			shouldRefresh, reason := needsRefresh(stylePath, len(transcripts), cfg.Refresh)
			if !shouldRefresh {
				fmt.Printf("Profile is up to date: %s\n", reason)
				return nil
			}
			fmt.Printf("Refreshing: %s\n", reason)
		} else {
			fmt.Println("Forcing refresh...")
		}

		fmt.Printf("Analyzing %d transcripts...\n", len(transcripts))

		profile, err := analyzer.Analyze(transcripts, cfg.LLM.Command, cfg.LLM.Args)
		if err != nil {
			return fmt.Errorf("analysis failed: %w", err)
		}

		if err := analyzer.SaveProfile(profile, profileDir); err != nil {
			return fmt.Errorf("saving profile: %w", err)
		}

		fmt.Printf("\nProfile updated: %d samples, %d words\n", profile.SampleCount, profile.TotalWords)
		return nil
	},
}

// needsRefresh checks whether the profile should be re-analyzed.
func needsRefresh(stylePath string, currentTranscripts int, cfg config.RefreshConfig) (bool, string) {
	data, err := os.ReadFile(stylePath)
	if err != nil {
		return true, "no existing profile found"
	}

	var existing struct {
		AnalyzedAt  string `json:"analyzed_at"`
		SampleCount int    `json:"sample_count"`
	}
	if err := json.Unmarshal(data, &existing); err != nil {
		return true, "existing profile is invalid"
	}

	// Check age
	analyzedAt, err := time.Parse(time.RFC3339, existing.AnalyzedAt)
	if err != nil {
		return true, "cannot parse profile timestamp"
	}

	minInterval, err := time.ParseDuration(cfg.MinInterval)
	if err != nil {
		minInterval = 24 * time.Hour
	}

	age := time.Since(analyzedAt)
	if age < minInterval {
		newCount := currentTranscripts - existing.SampleCount
		if newCount < cfg.MinNewTranscripts {
			return false, fmt.Sprintf("profile is %.0fh old, %d new transcripts (threshold: %d)",
				age.Hours(), newCount, cfg.MinNewTranscripts)
		}
	}

	// Check growth
	newTranscripts := currentTranscripts - existing.SampleCount
	if newTranscripts <= 0 {
		return false, "no new transcripts since last analysis"
	}

	// Guard division by zero: if SampleCount is 0, always refresh
	if existing.SampleCount == 0 {
		return true, fmt.Sprintf("%d new transcripts (no previous samples)", newTranscripts)
	}

	// 10% growth threshold
	growthPct := float64(newTranscripts) / float64(existing.SampleCount) * 100
	if newTranscripts >= cfg.MinNewTranscripts || growthPct >= 10 {
		return true, fmt.Sprintf("%d new transcripts (%.0f%% growth)", newTranscripts, growthPct)
	}

	return false, fmt.Sprintf("only %d new transcripts (%.0f%% growth, need 10%% or %d)",
		newTranscripts, growthPct, cfg.MinNewTranscripts)
}

func init() {
	refreshCmd.Flags().Bool("force", false, "force re-analysis regardless of thresholds")
	rootCmd.AddCommand(refreshCmd)
}
