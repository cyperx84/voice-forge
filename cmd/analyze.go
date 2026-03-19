package cmd

import (
	"fmt"

	"github.com/cyperx84/voice-forge/internal/analyzer"
	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/corpus"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Extract a comprehensive style profile from your voice corpus",
	Long: `Reads all transcripts from your voice corpus directories, sends them
to an LLM for deep style analysis, and outputs a structured profile.

Output:
  ~/.forge/profile/style.json       — machine-readable style profile
  ~/.forge/profile/style-summary.md — human-readable summary

Example:
  forge analyze`,
	RunE: func(cmd *cobra.Command, args []string) error {
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
			return fmt.Errorf("no transcripts found in corpus directories: %v", paths)
		}

		fmt.Printf("Found %d transcripts across %d corpus directories\n", len(transcripts), len(paths))

		profile, err := analyzer.Analyze(transcripts, cfg.LLM.Command, cfg.LLM.Args)
		if err != nil {
			return fmt.Errorf("analysis failed: %w", err)
		}

		outputDir := cfg.ProfileDir()
		if err := analyzer.SaveProfile(profile, outputDir); err != nil {
			return fmt.Errorf("saving profile: %w", err)
		}

		fmt.Println("\nStyle extraction complete.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(analyzeCmd)
}
