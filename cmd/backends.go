package cmd

import (
	"fmt"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/tts"
	"github.com/spf13/cobra"
)

// backendInfo holds display metadata for each backend.
type backendInfo struct {
	name        string
	description string
	installHint string
}

var knownBackends = []backendInfo{
	{"chatterbox", "Chatterbox Turbo 350M", "pip3 install chatterbox-tts"},
	{"f5-tts", "F5-TTS zero-shot cloning", "pip3 install f5-tts"},
	{"kokoro", "Kokoro (via tts-toolkit)", "configure [tts.tts_toolkit] in ~/.forge/config.toml"},
	{"elevenlabs", "ElevenLabs API", "set api_key in ~/.forge/config.toml [tts.elevenlabs]"},
	{"tts-toolkit", "tts-toolkit CLI wrapper", "pip3 install soundfile numpy torch"},
}

var backendsCmd = &cobra.Command{
	Use:   "backends",
	Short: "List all TTS backends and their availability",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		initBackends(cfg)

		fmt.Println("TTS Backends")
		fmt.Println()

		for _, info := range knownBackends {
			b, err := tts.Get(info.name)
			if err != nil {
				fmt.Printf("  %-14s  ❌ not registered\n", info.name)
				continue
			}

			if b.Available() {
				fmt.Printf("  %-14s  ✅ available (%s)\n", info.name, info.description)
			} else {
				fmt.Printf("  %-14s  ❌ not installed (%s)\n", info.name, info.installHint)
			}
		}

		fmt.Printf("\nDefault backend: %s\n", cfg.TTS.DefaultBackend)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(backendsCmd)
}
