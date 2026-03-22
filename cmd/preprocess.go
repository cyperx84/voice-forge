package cmd

import (
	"fmt"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/ffmpeg"
	"github.com/cyperx84/voice-forge/internal/preprocess"
	"github.com/spf13/cobra"
)

var (
	preprocessInput  string
	preprocessOutput string
	preprocessForce  bool
)

var preprocessCmd = &cobra.Command{
	Use:   "preprocess",
	Short: "Normalize and clean corpus audio for TTS training",
	Long: `Preprocess audio files: normalize format (24kHz mono 16-bit WAV),
denoise with ffmpeg afftdn, and segment on silence gaps.

Examples:
  forge preprocess
  forge preprocess --input ~/corpus --output ~/processed --force`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load config: %w", err)
		}

		input := preprocessInput
		if input == "" {
			paths := cfg.CorpusPaths()
			if len(paths) > 0 {
				input = paths[0]
			} else {
				input = config.ExpandPath("~/.forge/corpus")
			}
		} else {
			input = config.ExpandPath(input)
		}

		output := preprocessOutput
		if output == "" {
			output = config.ExpandPath("~/.forge/processed")
		} else {
			output = config.ExpandPath(output)
		}

		fmt.Printf("Preprocessing %s → %s\n", input, output)

		ffCfg := ffmpeg.Config{Threads: cfg.FFmpeg.Threads, Nice: cfg.FFmpeg.Nice}
		manifest, err := preprocess.Run(input, output, preprocessForce, cfg.Preprocess, ffCfg)
		if err != nil {
			return fmt.Errorf("preprocess: %w", err)
		}

		totalSegments := 0
		for _, f := range manifest.Files {
			totalSegments += len(f.Segments)
		}

		fmt.Printf("Processed %d files → %d segments\n", len(manifest.Files), totalSegments)
		return nil
	},
}

func init() {
	preprocessCmd.Flags().StringVar(&preprocessInput, "input", "", "input directory (default: corpus path from config)")
	preprocessCmd.Flags().StringVar(&preprocessOutput, "output", "", "output directory (default: ~/.forge/processed)")
	preprocessCmd.Flags().BoolVar(&preprocessForce, "force", false, "reprocess already-processed files")
	rootCmd.AddCommand(preprocessCmd)
}
