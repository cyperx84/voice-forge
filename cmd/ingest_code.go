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

var ingestCodeCmd = &cobra.Command{
	Use:   "ingest-code [path...]",
	Short: "Ingest code samples to capture coding style",
	Long: `Ingest code files or directories to analyze coding style.

Examples:
  forge ingest-code ~/github/voice-forge/ --language go
  forge ingest-code main.go utils.go --source voice-forge`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		source, _ := cmd.Flags().GetString("source")
		language, _ := cmd.Flags().GetString("language")
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

		opts := ingest.CodeOptions{
			Source:   source,
			Language: language,
			Tags:     tags,
		}

		total := 0
		for _, arg := range args {
			path := config.ExpandPath(arg)
			info, err := os.Stat(path)
			if err != nil {
				fmt.Printf("  error: %s: %v\n", arg, err)
				continue
			}

			if info.IsDir() {
				items, err := ingest.IngestCodeDir(db, cfg.CorpusRoot(), path, opts)
				if err != nil {
					fmt.Printf("  error scanning %s: %v\n", arg, err)
					continue
				}
				fmt.Printf("  %s: %d file(s)\n", arg, len(items))
				total += len(items)
			} else {
				_, err := ingest.IngestCodeFile(db, cfg.CorpusRoot(), path, opts)
				if err != nil {
					fmt.Printf("  error: %s: %v\n", arg, err)
					continue
				}
				total++
				fmt.Printf("  ingested: %s\n", arg)
			}
		}

		fmt.Printf("Ingested %d code file(s)\n", total)
		return nil
	},
}

func init() {
	ingestCodeCmd.Flags().String("source", "", "source label (repo name, project, etc.)")
	ingestCodeCmd.Flags().String("language", "", "filter by language (go, python, typescript, etc.)")
	ingestCodeCmd.Flags().String("tags", "", "comma-separated tags")
	rootCmd.AddCommand(ingestCodeCmd)
}
