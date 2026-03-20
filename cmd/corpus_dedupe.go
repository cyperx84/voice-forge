package cmd

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/corpus"
	"github.com/spf13/cobra"
)

var corpusDedupeCmd = &cobra.Command{
	Use:   "dedupe",
	Short: "Detect and remove duplicate corpus entries",
	Long:  `Finds duplicates by content hash or file path and removes them, keeping the earliest entry.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		db, err := corpus.OpenDB(cfg.CorpusDBPath())
		if err != nil {
			return fmt.Errorf("opening corpus db: %w", err)
		}
		defer db.Close()

		dryRun, _ := cmd.Flags().GetBool("dry-run")

		items, err := db.ExportAll()
		if err != nil {
			return fmt.Errorf("loading items: %w", err)
		}

		// Track seen content hashes and paths
		seenHash := map[string]string{} // hash -> first ID
		seenPath := map[string]string{} // path -> first ID
		var dupeIDs []string

		for _, item := range items {
			// Check path duplicates
			if item.Path != "" {
				if firstID, ok := seenPath[item.Path]; ok {
					if item.ID != firstID {
						dupeIDs = append(dupeIDs, item.ID)
						continue
					}
				} else {
					seenPath[item.Path] = item.ID
				}
			}

			// Check content duplicates
			if item.Transcript != "" {
				h := sha256.Sum256([]byte(strings.TrimSpace(item.Transcript)))
				hash := fmt.Sprintf("%x", h)
				if firstID, ok := seenHash[hash]; ok {
					if item.ID != firstID {
						dupeIDs = append(dupeIDs, item.ID)
						continue
					}
				} else {
					seenHash[hash] = item.ID
				}
			}
		}

		if len(dupeIDs) == 0 {
			fmt.Println("No duplicates found.")
			return nil
		}

		if dryRun {
			fmt.Printf("Found %d duplicate(s) (dry run, not removing):\n", len(dupeIDs))
			for _, id := range dupeIDs {
				fmt.Printf("  %s\n", id)
			}
			return nil
		}

		removed, err := db.DeleteByIDs(dupeIDs)
		if err != nil {
			return fmt.Errorf("removing duplicates: %w", err)
		}

		fmt.Printf("Removed %d duplicate(s)\n", removed)
		return nil
	},
}

func init() {
	corpusDedupeCmd.Flags().Bool("dry-run", false, "show duplicates without removing them")
	corpusCmd.AddCommand(corpusDedupeCmd)
}
