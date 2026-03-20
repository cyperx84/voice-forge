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
	Short: "Extract a comprehensive style profile from your corpus",
	Long: `Reads all content from your corpus (voice transcripts, text, code, social posts),
sends them to an LLM for deep style analysis, and outputs a structured profile.

With --multi-source (default when corpus DB exists), analyzes all corpus types:
  - Voice style (from voice transcripts)
  - Writing style (from text corpus)
  - Coding style (from code corpus)
  - Content themes (across all sources)

Output:
  ~/.forge/profile/style.json       — machine-readable style profile
  ~/.forge/profile/style-summary.md — human-readable summary

Example:
  forge analyze
  forge analyze --voice-only`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		voiceOnly, _ := cmd.Flags().GetBool("voice-only")

		// Always read voice transcripts from filesystem (backward compatible)
		paths := cfg.CorpusPaths()
		voiceTranscripts, err := corpus.ReadTranscripts(paths)
		if err != nil {
			return fmt.Errorf("reading transcripts: %w", err)
		}

		outputDir := cfg.ProfileDir()

		// Try multi-source analysis if corpus DB exists
		if !voiceOnly {
			db, dbErr := corpus.OpenDB(cfg.CorpusDBPath())
			if dbErr == nil {
				defer db.Close()

				// Auto-migrate existing voice files
				corpus.MigrateExistingCorpus(paths, db)

				textSamples, _ := db.AllTranscripts("text")
				codeSamples, _ := db.AllTranscripts("code")
				socialSamples, _ := db.AllTranscripts("social")

				hasExtra := len(textSamples) > 0 || len(codeSamples) > 0 || len(socialSamples) > 0
				if hasExtra || len(voiceTranscripts) > 0 {
					fmt.Printf("Multi-source analysis: %d voice, %d text, %d code, %d social\n",
						len(voiceTranscripts), len(textSamples), len(codeSamples), len(socialSamples))

					profile, err := analyzer.AnalyzeMultiSource(
						voiceTranscripts, textSamples, codeSamples, socialSamples,
						cfg.LLM.Command, cfg.LLM.Args,
					)
					if err != nil {
						return fmt.Errorf("multi-source analysis failed: %w", err)
					}

					if err := analyzer.SaveMultiSourceProfile(profile, outputDir); err != nil {
						return fmt.Errorf("saving profile: %w", err)
					}

					fmt.Println("\nMulti-source style extraction complete.")
					return nil
				}
			}
		}

		// Fallback to voice-only analysis
		if len(voiceTranscripts) == 0 {
			return fmt.Errorf("no transcripts found in corpus directories: %v", paths)
		}

		fmt.Printf("Found %d transcripts across %d corpus directories\n", len(voiceTranscripts), len(paths))

		profile, err := analyzer.Analyze(voiceTranscripts, cfg.LLM.Command, cfg.LLM.Args)
		if err != nil {
			return fmt.Errorf("analysis failed: %w", err)
		}

		if err := analyzer.SaveProfile(profile, outputDir); err != nil {
			return fmt.Errorf("saving profile: %w", err)
		}

		fmt.Println("\nStyle extraction complete.")
		return nil
	},
}

func init() {
	analyzeCmd.Flags().Bool("voice-only", false, "only analyze voice transcripts (skip multi-source)")
	rootCmd.AddCommand(analyzeCmd)
}
