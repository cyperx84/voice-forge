package cmd

import (
	"fmt"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/ffmpeg"
	"github.com/cyperx84/voice-forge/internal/scoring"
	"github.com/spf13/cobra"
)

var (
	scoreInput     string
	scoreThreshold string
)

var scoreCmd = &cobra.Command{
	Use:   "score",
	Short: "Quality-score each recording or segment",
	Long: `Score audio files by SNR, clipping, duration, and silence ratio.
Assigns tiers: gold, silver, bronze, or reject.

Examples:
  forge score
  forge score --input ~/.forge/processed --threshold gold`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		input := scoreInput
		if input == "" {
			input = config.ExpandPath("~/.forge/processed")
		} else {
			input = config.ExpandPath(input)
		}

		threshold := scoreThreshold
		if threshold == "" {
			threshold = cfg.Scoring.DefaultThreshold
		}

		fmt.Printf("Scoring files in %s (threshold: %s)\n", input, threshold)

		ffCfg := ffmpeg.Config{Threads: cfg.FFmpeg.Threads, Nice: cfg.FFmpeg.Nice}
		report, err := scoring.ScoreDir(input, ffCfg)
		if err != nil {
			return fmt.Errorf("score: %w", err)
		}

		outputPath := config.ExpandPath("~/.forge/scores.json")
		if err := scoring.SaveReport(report, outputPath); err != nil {
			return fmt.Errorf("save report: %w", err)
		}

		total := len(report.Files)
		fmt.Printf("%d files scored: %d Gold, %d Silver, %d Bronze, %d Reject\n",
			total, report.Gold, report.Silver, report.Bronze, report.Reject)

		return nil
	},
}

func init() {
	scoreCmd.Flags().StringVar(&scoreInput, "input", "", "input directory (default: ~/.forge/processed)")
	scoreCmd.Flags().StringVar(&scoreThreshold, "threshold", "", "minimum tier threshold (default: from config)")
	rootCmd.AddCommand(scoreCmd)
}
