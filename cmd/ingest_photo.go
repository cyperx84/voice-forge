package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/corpus"
	"github.com/cyperx84/voice-forge/internal/ingest"
	"github.com/spf13/cobra"
)

var ingestPhotoCmd = &cobra.Command{
	Use:   "ingest-photo [path]",
	Short: "Ingest photos into the corpus",
	Long: `Ingest photos with optional tagging.

Examples:
  forge ingest-photo ~/Photos/headshot.jpg --tags "profile,headshot"
  forge ingest-photo ~/Photos/brand/ --source brand-kit --recursive`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		source, _ := cmd.Flags().GetString("source")
		tagsStr, _ := cmd.Flags().GetString("tags")
		recursive, _ := cmd.Flags().GetBool("recursive")

		var tags []string
		if tagsStr != "" {
			tags = strings.Split(tagsStr, ",")
		}

		db, err := corpus.OpenDB(cfg.CorpusDBPath())
		if err != nil {
			return fmt.Errorf("opening corpus db: %w", err)
		}
		defer db.Close()

		path := config.ExpandPath(args[0])
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("stat: %w", err)
		}

		opts := ingest.PhotoOptions{
			Source:    source,
			Tags:      tags,
			Recursive: recursive,
		}

		if info.IsDir() {
			items, err := ingest.IngestPhotoDir(db, cfg.CorpusRoot(), path, opts)
			if err != nil {
				return fmt.Errorf("ingesting photos: %w", err)
			}
			fmt.Printf("Ingested %d photo(s) from %s\n", len(items), path)
		} else {
			_, err := ingest.IngestPhotoFile(db, cfg.CorpusRoot(), path, opts)
			if err != nil {
				return fmt.Errorf("ingesting photo: %w", err)
			}
			fmt.Printf("Ingested photo: %s\n", path)
		}
		return nil
	},
}

func init() {
	ingestPhotoCmd.Flags().String("source", "", "source label")
	ingestPhotoCmd.Flags().String("tags", "", "comma-separated tags")
	ingestPhotoCmd.Flags().Bool("recursive", false, "recursively ingest directory")
	rootCmd.AddCommand(ingestPhotoCmd)
}
