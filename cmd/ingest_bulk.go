package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/corpus"
	"github.com/cyperx84/voice-forge/internal/ingest"
	"github.com/spf13/cobra"
)

var ingestBulkVoiceDirs []string
var ingestBulkCodeDirs []string
var ingestBulkTextDirs []string
var ingestBulkAll bool
var ingestBulkDryRun bool

var ingestBulkCmd = &cobra.Command{
	Use:   "ingest-bulk",
	Short: "Bulk ingest voice, code, and text from multiple directories",
	Long: `Ingest everything at once from configured or specified directories.

Examples:
  forge ingest-bulk --voice ~/voice-corpus/ ~/voice/
  forge ingest-bulk --code ~/github/voice-forge/ ~/github/tts-toolkit/
  forge ingest-bulk --text ~/documents/notes/
  forge ingest-bulk --all
  forge ingest-bulk --all --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// If --all, use configured paths
		if ingestBulkAll {
			if len(ingestBulkVoiceDirs) == 0 {
				ingestBulkVoiceDirs = cfg.Ingest.BulkVoiceDirs
				if len(ingestBulkVoiceDirs) == 0 {
					// Fall back to corpus paths
					ingestBulkVoiceDirs = cfg.Corpus.Paths
				}
			}
			if len(ingestBulkCodeDirs) == 0 {
				ingestBulkCodeDirs = cfg.Ingest.BulkCodeDirs
			}
			if len(ingestBulkTextDirs) == 0 {
				ingestBulkTextDirs = cfg.Ingest.BulkTextDirs
			}
		}

		if len(ingestBulkVoiceDirs) == 0 && len(ingestBulkCodeDirs) == 0 && len(ingestBulkTextDirs) == 0 {
			return fmt.Errorf("no directories specified — use --voice, --code, --text, or --all\n" +
				"configure default paths in ~/.forge/config.toml under [ingest]:\n" +
				"  bulk_voice_dirs = [\"~/voice-corpus\"]\n" +
				"  bulk_code_dirs = [\"~/github/myproject\"]\n" +
				"  bulk_text_dirs = [\"~/notes\"]")
		}

		if ingestBulkDryRun {
			return dryRunBulk(cfg)
		}

		db, err := corpus.OpenDB(cfg.CorpusDBPath())
		if err != nil {
			return fmt.Errorf("opening corpus db: %w", err)
		}
		defer db.Close()

		corpusRoot := cfg.CorpusRoot()
		totalIngested := 0

		// Ingest voice directories
		for _, dir := range ingestBulkVoiceDirs {
			dir = config.ExpandPath(dir)
			fmt.Printf("Voice: %s\n", dir)
			count, err := ingestVoiceDir(db, dir)
			if err != nil {
				fmt.Printf("  error: %v\n", err)
				continue
			}
			fmt.Printf("  %d item(s) ingested\n", count)
			totalIngested += count
		}

		// Ingest code directories
		for _, dir := range ingestBulkCodeDirs {
			dir = config.ExpandPath(dir)
			fmt.Printf("Code: %s\n", dir)
			items, err := ingest.IngestCodeDir(db, corpusRoot, dir, ingest.CodeOptions{
				Source: filepath.Base(dir),
			})
			if err != nil {
				fmt.Printf("  error: %v\n", err)
				continue
			}
			fmt.Printf("  %d file(s) ingested\n", len(items))
			totalIngested += len(items)
		}

		// Ingest text directories
		for _, dir := range ingestBulkTextDirs {
			dir = config.ExpandPath(dir)
			fmt.Printf("Text: %s\n", dir)
			count, err := ingestTextDir(db, corpusRoot, dir)
			if err != nil {
				fmt.Printf("  error: %v\n", err)
				continue
			}
			fmt.Printf("  %d file(s) ingested\n", count)
			totalIngested += count
		}

		fmt.Printf("\nTotal: %d item(s) ingested\n", totalIngested)
		return nil
	},
}

func dryRunBulk(cfg config.Config) error {
	fmt.Println("Dry run — showing what would be ingested:")
	fmt.Println()

	for _, dir := range ingestBulkVoiceDirs {
		dir = config.ExpandPath(dir)
		fmt.Printf("Voice: %s\n", dir)
		count := countVoiceFiles(dir)
		fmt.Printf("  %d voice file(s) found\n", count)
	}

	for _, dir := range ingestBulkCodeDirs {
		dir = config.ExpandPath(dir)
		fmt.Printf("Code: %s\n", dir)
		count := countCodeFiles(dir)
		fmt.Printf("  %d code file(s) found\n", count)
	}

	for _, dir := range ingestBulkTextDirs {
		dir = config.ExpandPath(dir)
		fmt.Printf("Text: %s\n", dir)
		count := countTextFiles(dir)
		fmt.Printf("  %d text file(s) found\n", count)
	}

	return nil
}

func ingestVoiceDir(db *corpus.DB, dir string) (int, error) {
	migrated, err := corpus.MigrateExistingCorpus([]string{dir}, db)
	return migrated, err
}

func ingestTextDir(db *corpus.DB, corpusRoot, dir string) (int, error) {
	count := 0
	skipDirs := map[string]bool{".git": true, "node_modules": true, ".venv": true}

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(info.Name()))
		if ext != ".md" && ext != ".txt" {
			return nil
		}
		if info.Size() > 100*1024 {
			return nil
		}

		_, err = ingest.IngestTextFile(db, corpusRoot, path, ingest.TextOptions{
			Source: filepath.Base(dir),
		})
		if err != nil {
			return nil
		}
		count++
		return nil
	})
	return count, err
}

func countVoiceFiles(dir string) int {
	count := 0
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext == ".ogg" || ext == ".wav" || ext == ".txt" {
			count++
		}
	}
	// Also check transcripts/ subdirectory
	subEntries, err := os.ReadDir(filepath.Join(dir, "transcripts"))
	if err == nil {
		for _, e := range subEntries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".txt") {
				count++
			}
		}
	}
	return count
}

func countCodeFiles(dir string) int {
	count := 0
	skipDirs := map[string]bool{
		".git": true, "node_modules": true, "vendor": true, "__pycache__": true,
		".venv": true, "target": true, "build": true, "dist": true, ".next": true,
	}
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if ingest.IsCodeFile(info.Name()) && info.Size() <= 100*1024 {
			count++
		}
		return nil
	})
	return count
}

func countTextFiles(dir string) int {
	count := 0
	skipDirs := map[string]bool{".git": true, "node_modules": true, ".venv": true}
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(info.Name()))
		if (ext == ".md" || ext == ".txt") && info.Size() <= 100*1024 {
			count++
		}
		return nil
	})
	return count
}

func init() {
	ingestBulkCmd.Flags().StringSliceVar(&ingestBulkVoiceDirs, "voice", nil, "voice corpus directories to ingest")
	ingestBulkCmd.Flags().StringSliceVar(&ingestBulkCodeDirs, "code", nil, "code directories to ingest")
	ingestBulkCmd.Flags().StringSliceVar(&ingestBulkTextDirs, "text", nil, "text/markdown directories to ingest")
	ingestBulkCmd.Flags().BoolVar(&ingestBulkAll, "all", false, "use all configured paths from config.toml")
	ingestBulkCmd.Flags().BoolVar(&ingestBulkDryRun, "dry-run", false, "show what would be ingested without doing it")
	rootCmd.AddCommand(ingestBulkCmd)
}
