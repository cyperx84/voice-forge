package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/corpus"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Display corpus statistics",
	Long: `Shows statistics about your voice corpus including total recordings,
duration, word counts, and most frequently used words.

Example:
  forge stats`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		paths := cfg.CorpusPaths()

		// Parse manifest for duration data
		var allRecordings []corpus.Recording
		for _, p := range paths {
			manifestPath := filepath.Join(p, "manifest.txt")
			recs, err := corpus.ParseManifest(manifestPath)
			if err != nil {
				continue
			}
			allRecordings = append(allRecordings, recs...)
		}

		// Read transcripts
		transcripts, err := corpus.ReadTranscripts(paths)
		if err != nil {
			return fmt.Errorf("reading transcripts: %w", err)
		}

		stats := corpus.ComputeStats(allRecordings, transcripts)

		// Get date range from file modification times
		earliest, latest := corpus.GetFileModTimes(paths)

		// Display
		fmt.Println("Voice Corpus Statistics")
		fmt.Println(strings.Repeat("═", 45))
		fmt.Printf("  Recordings:    %d\n", stats.TotalRecordings)
		fmt.Printf("  Total duration: %s\n", formatDuration(stats.TotalDuration))
		fmt.Printf("  Avg duration:   %s\n", formatDuration(stats.AvgDuration))
		if !earliest.IsZero() {
			fmt.Printf("  Date range:     %s → %s\n", earliest.Format("2006-01-02"), latest.Format("2006-01-02"))
		}
		fmt.Println()
		fmt.Printf("  Total words:    %d\n", stats.TotalWords)
		fmt.Printf("  Unique words:   %d\n", stats.UniqueWords)
		fmt.Println()

		if len(stats.TopWords) > 0 {
			fmt.Println("Top 50 Words (excluding stop words)")
			fmt.Println(strings.Repeat("─", 45))
			for i, wc := range stats.TopWords {
				fmt.Printf("  %2d. %-20s %d\n", i+1, wc.Word, wc.Count)
			}
		}

		return nil
	},
}

func formatDuration(d interface{ Hours() float64; Minutes() float64; Seconds() float64 }) string {
	type dur interface {
		Hours() float64
		Minutes() float64
		Seconds() float64
	}
	return formatDur(d)
}

func formatDur(d interface{ Hours() float64; Minutes() float64; Seconds() float64 }) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	}
	return fmt.Sprintf("%ds", seconds)
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
