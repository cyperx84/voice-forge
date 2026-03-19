package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/export"
	"github.com/cyperx84/voice-forge/internal/scoring"
	"github.com/spf13/cobra"
)

var (
	exportFormat string
	exportTier   string
	exportOutput string
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export corpus in standard TTS training formats",
	Long: `Export scored audio files in LJSpeech or JSONL format,
filtered by quality tier.

Examples:
  forge export --format ljspeech --tier gold,silver
  forge export --format jsonl --output ~/export/`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		format := exportFormat
		if format == "" {
			format = cfg.Export.DefaultFormat
		}

		tierStr := exportTier
		if tierStr == "" {
			tierStr = cfg.Export.DefaultTier
		}
		threshold := scoring.Tier(tierStr)

		output := exportOutput
		if output == "" {
			output = config.ExpandPath("~/.forge/export")
		} else {
			output = config.ExpandPath(output)
		}

		// Load score report
		scoresPath := config.ExpandPath("~/.forge/scores.json")
		data, err := os.ReadFile(scoresPath)
		if err != nil {
			return fmt.Errorf("read scores (run 'forge score' first): %w", err)
		}
		var report scoring.Report
		if err := json.Unmarshal(data, &report); err != nil {
			return fmt.Errorf("parse scores: %w", err)
		}

		transcriptDir := config.ExpandPath("~/.forge/processed")

		var count int
		switch format {
		case "ljspeech":
			count, err = export.ExportLJSpeech(&report, transcriptDir, output, threshold)
		case "jsonl":
			outputPath := fmt.Sprintf("%s/export.jsonl", output)
			count, err = export.ExportJSONL(&report, transcriptDir, outputPath, threshold)
		default:
			return fmt.Errorf("unsupported format: %s (use ljspeech or jsonl)", format)
		}

		if err != nil {
			return fmt.Errorf("export: %w", err)
		}

		fmt.Printf("Exported %d files in %s format (tier >= %s) → %s\n", count, format, tierStr, output)
		return nil
	},
}

func init() {
	exportCmd.Flags().StringVar(&exportFormat, "format", "", "export format: ljspeech or jsonl (default: from config)")
	exportCmd.Flags().StringVar(&exportTier, "tier", "", "minimum tier: gold, silver, bronze (default: from config)")
	exportCmd.Flags().StringVar(&exportOutput, "output", "", "output directory (default: ~/.forge/export)")
	rootCmd.AddCommand(exportCmd)
}
