package cmd

import (
	"fmt"
	"strings"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/corpus"
	"github.com/cyperx84/voice-forge/internal/ingest"
	"github.com/spf13/cobra"
)

var ingestVideoCmd = &cobra.Command{
	Use:   "ingest-video [file]",
	Short: "Ingest video content into the corpus",
	Long: `Ingest video files. Extracts: transcript (whisper), keyframes, metadata.

Examples:
  forge ingest-video ~/Videos/talk.mp4 --source local
  forge ingest-video ~/clip.mp4 --transcript-only`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		source, _ := cmd.Flags().GetString("source")
		transcriptOnly, _ := cmd.Flags().GetBool("transcript-only")
		tagsStr, _ := cmd.Flags().GetString("tags")

		var tags []string
		if tagsStr != "" {
			tags = strings.Split(tagsStr, ",")
		}

		db, err := corpus.OpenDB(cfg.CorpusDBPath())
		if err != nil {
			return fmt.Errorf("opening corpus db: %w", err)
		}
		defer db.Close()

		whisperCmd := cfg.Ingest.WhisperCommand
		if whisperCmd == "" {
			whisperCmd = cfg.Watch.WhisperCommand
		}

		path := config.ExpandPath(args[0])
		item, err := ingest.IngestVideoFile(db, cfg.CorpusRoot(), path, ingest.VideoOptions{
			Source:           source,
			TranscriptOnly:   transcriptOnly,
			WhisperCommand:   whisperCmd,
			WhisperModel:     cfg.Watch.WhisperModel,
			KeyframeInterval: cfg.Ingest.VideoKeyframeInterval,
			Tags:             tags,
		})
		if err != nil {
			return fmt.Errorf("ingesting video: %w", err)
		}

		fmt.Printf("Ingested video: %s\n", path)
		if item.Transcript != "" {
			fmt.Printf("  transcript: %d words\n", item.WordCount)
		}
		if item.DurationSeconds > 0 {
			fmt.Printf("  duration: %.1fs\n", item.DurationSeconds)
		}
		return nil
	},
}

func init() {
	ingestVideoCmd.Flags().String("source", "", "source label")
	ingestVideoCmd.Flags().Bool("transcript-only", false, "only extract transcript, skip keyframes")
	ingestVideoCmd.Flags().String("tags", "", "comma-separated tags")
	rootCmd.AddCommand(ingestVideoCmd)
}
