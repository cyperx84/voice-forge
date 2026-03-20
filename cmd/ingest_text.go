package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/corpus"
	"github.com/cyperx84/voice-forge/internal/ingest"
	"github.com/spf13/cobra"
)

var ingestTextCmd = &cobra.Command{
	Use:   "ingest-text [file]",
	Short: "Ingest written content into the corpus",
	Long: `Ingest text files (markdown, plain text) or structured exports (Discord, Twitter) into the corpus.

Examples:
  forge ingest-text ~/blog/my-post.md --source blog
  forge ingest-text ~/discord-export.json --source discord --format discord-export
  forge ingest-text ~/tweets.json --source twitter --format twitter-archive
  echo "some text" | forge ingest-text --source note`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		source, _ := cmd.Flags().GetString("source")
		format, _ := cmd.Flags().GetString("format")
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

		corpusRoot := cfg.CorpusRoot()

		// Handle structured formats
		if format != "" && len(args) > 0 {
			switch format {
			case "discord-export":
				posts, err := ingest.ParseDiscordExport(args[0])
				if err != nil {
					return err
				}
				items, err := ingest.IngestSocialPosts(db, corpusRoot, posts, "discord")
				if err != nil {
					return err
				}
				fmt.Printf("Ingested %d Discord messages\n", len(items))
				return nil

			case "twitter-archive":
				posts, err := ingest.ParseTwitterArchive(args[0])
				if err != nil {
					return err
				}
				items, err := ingest.IngestSocialPosts(db, corpusRoot, posts, "twitter")
				if err != nil {
					return err
				}
				fmt.Printf("Ingested %d tweets\n", len(items))
				return nil

			default:
				return fmt.Errorf("unknown format: %s (supported: discord-export, twitter-archive)", format)
			}
		}

		// Handle file argument
		if len(args) > 0 {
			for _, f := range args {
				path := config.ExpandPath(f)
				item, err := ingest.IngestTextFile(db, corpusRoot, path, ingest.TextOptions{
					Source: source,
					Tags:   tags,
				})
				if err != nil {
					fmt.Printf("  error: %s: %v\n", f, err)
					continue
				}
				fmt.Printf("  ingested: %s (%d words)\n", f, item.WordCount)
			}
			return nil
		}

		// Handle stdin
		stat, _ := os.Stdin.Stat()
		if stat.Mode()&os.ModeCharDevice == 0 {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("reading stdin: %w", err)
			}
			item, err := ingest.IngestTextString(db, corpusRoot, string(data), source, tags)
			if err != nil {
				return err
			}
			fmt.Printf("Ingested from stdin (%d words)\n", item.WordCount)
			return nil
		}

		return fmt.Errorf("provide a file path or pipe text to stdin")
	},
}

func init() {
	ingestTextCmd.Flags().String("source", "", "source label (blog, discord, note, etc.)")
	ingestTextCmd.Flags().String("format", "", "structured format (discord-export, twitter-archive)")
	ingestTextCmd.Flags().String("tags", "", "comma-separated tags")
	rootCmd.AddCommand(ingestTextCmd)
}
