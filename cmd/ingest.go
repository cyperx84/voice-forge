package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/spf13/cobra"
)

var ingestCmd = &cobra.Command{
	Use:   "ingest <path>",
	Short: "Import audio and transcript files into the voice corpus",
	Long: `Copies audio (.ogg, .wav) and transcript (.txt) files into the
primary voice corpus directory. Files should be named with matching
base names (e.g., recording.ogg + recording.txt).

Examples:
  forge ingest ./my-recording.ogg
  forge ingest ./recordings/`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// Use first corpus path as the destination
		if len(cfg.Corpus.Paths) == 0 {
			return fmt.Errorf("no corpus paths configured")
		}
		destDir := config.ExpandPath(cfg.Corpus.Paths[0])

		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("creating corpus directory: %w", err)
		}

		srcPath := args[0]
		info, err := os.Stat(srcPath)
		if err != nil {
			return fmt.Errorf("cannot access %s: %w", srcPath, err)
		}

		var files []string
		if info.IsDir() {
			entries, err := os.ReadDir(srcPath)
			if err != nil {
				return fmt.Errorf("reading directory: %w", err)
			}
			for _, e := range entries {
				if !e.IsDir() && isCorpusFile(e.Name()) {
					files = append(files, filepath.Join(srcPath, e.Name()))
				}
			}
		} else {
			if isCorpusFile(info.Name()) {
				files = append(files, srcPath)
			} else {
				return fmt.Errorf("unsupported file type: %s (expected .ogg, .wav, or .txt)", info.Name())
			}
		}

		if len(files) == 0 {
			return fmt.Errorf("no corpus files (.ogg, .wav, .txt) found at %s", srcPath)
		}

		copied := 0
		for _, f := range files {
			name := filepath.Base(f)
			dest := filepath.Join(destDir, name)

			if _, err := os.Stat(dest); err == nil {
				fmt.Printf("  skip (exists): %s\n", name)
				continue
			}

			if err := copyFile(f, dest); err != nil {
				fmt.Printf("  error: %s — %v\n", name, err)
				continue
			}
			fmt.Printf("  imported: %s\n", name)
			copied++
		}

		fmt.Printf("\nImported %d file(s) to %s\n", copied, destDir)
		return nil
	},
}

func isCorpusFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".ogg" || ext == ".wav" || ext == ".txt" || ext == ".mp3" || ext == ".m4a"
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// Write to temp file then rename for atomicity
	tmp, err := os.CreateTemp(filepath.Dir(dst), ".forge-ingest-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	if _, err = io.Copy(tmp, in); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return os.Rename(tmpPath, dst)
}

func init() {
	rootCmd.AddCommand(ingestCmd)
}
