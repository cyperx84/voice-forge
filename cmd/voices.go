package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/spf13/cobra"
)

var voicesCmd = &cobra.Command{
	Use:   "voices",
	Short: "List available cloned voices",
	Long: `Lists all cloned voices stored in ~/.forge/voices/.

Shows the voice name, associated backend, and number of samples used.

Example:
  forge voices`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		voicesDir := config.ExpandPath("~/.forge/voices")
		entries, err := os.ReadDir(voicesDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No cloned voices found. Use 'forge clone' to create one.")
				return nil
			}
			return fmt.Errorf("reading voices directory: %w", err)
		}

		if len(entries) == 0 {
			fmt.Println("No cloned voices found. Use 'forge clone' to create one.")
			return nil
		}

		defaultVoice := cfg.Voices.Default
		fmt.Println("Cloned voices:")
		fmt.Println()

		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			marker := "  "
			if name == defaultVoice {
				marker = "* "
			}

			// Read voice metadata if available
			metaPath := filepath.Join(voicesDir, name, "voice.toml")
			backend := "unknown"
			if data, err := os.ReadFile(metaPath); err == nil {
				backend = parseVoiceBackend(string(data))
			}

			fmt.Printf("%s%-16s  backend: %s\n", marker, name, backend)
		}

		if defaultVoice != "" {
			fmt.Printf("\n* = default voice\n")
		}

		return nil
	},
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func init() {
	rootCmd.AddCommand(voicesCmd)
}

func parseVoiceBackend(meta string) string {
	for _, line := range splitLines(meta) {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "backend = ") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(line, "backend = "))
		value = strings.TrimSpace(strings.Trim(value, "\"'"))
		if value != "" {
			return value
		}
	}
	return "unknown"
}
