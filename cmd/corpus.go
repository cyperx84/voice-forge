package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/corpus"
	"github.com/spf13/cobra"
)

var corpusCmd = &cobra.Command{
	Use:   "corpus",
	Short: "Unified corpus management",
	Long:  `Manage the multi-source identity corpus.`,
}

var corpusStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show corpus statistics",
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

		// Auto-migrate existing voice corpus
		migrated, _ := corpus.MigrateExistingCorpus(cfg.CorpusPaths(), db)
		if migrated > 0 {
			fmt.Printf("Migrated %d existing voice items\n\n", migrated)
		}

		itemType, _ := cmd.Flags().GetString("type")
		if itemType != "" {
			count, _ := db.Count(itemType)
			fmt.Printf("%s: %d item(s)\n", itemType, count)
			return nil
		}

		stats, err := db.Stats()
		if err != nil {
			return fmt.Errorf("getting stats: %w", err)
		}

		if len(stats) == 0 {
			fmt.Println("Corpus is empty. Use forge ingest-text, ingest-code, etc. to add content.")
			return nil
		}

		fmt.Println("Corpus Statistics")
		fmt.Println(strings.Repeat("─", 50))
		totalItems := 0
		totalWords := 0
		for _, s := range stats {
			fmt.Printf("  %-10s %5d items  %8d words", s.Type, s.Count, s.TotalWords)
			if s.TotalDur > 0 {
				fmt.Printf("  %.0fs audio", s.TotalDur)
			}
			if s.TotalSize > 0 {
				fmt.Printf("  %s", formatSize(s.TotalSize))
			}
			fmt.Println()
			totalItems += s.Count
			totalWords += s.TotalWords
		}
		fmt.Println(strings.Repeat("─", 50))
		fmt.Printf("  %-10s %5d items  %8d words\n", "TOTAL", totalItems, totalWords)

		return nil
	},
}

var corpusSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search across all corpus types",
	Args:  cobra.ExactArgs(1),
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

		items, err := db.Search(args[0])
		if err != nil {
			return fmt.Errorf("searching: %w", err)
		}

		if len(items) == 0 {
			fmt.Printf("No results for %q\n", args[0])
			return nil
		}

		fmt.Printf("Found %d result(s) for %q:\n\n", len(items), args[0])
		for _, item := range items {
			preview := item.Transcript
			if len(preview) > 100 {
				preview = preview[:100] + "..."
			}
			preview = strings.ReplaceAll(preview, "\n", " ")
			fmt.Printf("  [%s] %s (%s)\n", item.Type, item.ID[:8], item.Source)
			if preview != "" {
				fmt.Printf("    %s\n", preview)
			}
		}
		return nil
	},
}

var corpusRecentCmd = &cobra.Command{
	Use:   "recent",
	Short: "List recent additions",
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

		limit, _ := cmd.Flags().GetInt("limit")
		items, err := db.Recent(limit)
		if err != nil {
			return fmt.Errorf("listing recent: %w", err)
		}

		if len(items) == 0 {
			fmt.Println("No items in corpus.")
			return nil
		}

		for _, item := range items {
			preview := item.Transcript
			if len(preview) > 80 {
				preview = preview[:80] + "..."
			}
			preview = strings.ReplaceAll(preview, "\n", " ")
			fmt.Printf("  [%s] %s  %s  %s\n", item.Type, item.ID[:8], item.Source, item.IngestedAt[:10])
			if preview != "" {
				fmt.Printf("    %s\n", preview)
			}
		}
		return nil
	},
}

var corpusExportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export corpus manifest",
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

		items, err := db.ExportAll()
		if err != nil {
			return fmt.Errorf("exporting: %w", err)
		}

		data, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(data))
		return nil
	},
}

func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
}

func init() {
	corpusStatsCmd.Flags().String("type", "", "filter by type (voice, text, video, photo, code, social)")
	corpusRecentCmd.Flags().Int("limit", 20, "number of recent items to show")
	corpusExportCmd.Flags().String("format", "json", "export format")

	corpusCmd.AddCommand(corpusStatsCmd)
	corpusCmd.AddCommand(corpusSearchCmd)
	corpusCmd.AddCommand(corpusRecentCmd)
	corpusCmd.AddCommand(corpusExportCmd)
	rootCmd.AddCommand(corpusCmd)
}
