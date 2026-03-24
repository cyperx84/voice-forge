package cmd

import (
	"fmt"
	"strings"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/tts"
	"github.com/spf13/cobra"
)

// backendInfo holds display metadata for each backend.
type backendInfo struct {
	name        string
	description string
	installHint string
	configHint  string
}

var knownBackends = []backendInfo{
	{"chatterbox", "Chatterbox local voice clone", "pip3 install chatterbox-tts", "uses ~/.forge/venv if present"},
	{"f5-tts", "F5-TTS zero-shot cloning", "pip3 install f5-tts", "uses ~/.forge/venv if present"},
	{"kokoro", "Kokoro (via tts-toolkit)", "configure [tts.tts_toolkit] in ~/.forge/config.toml", "runtime depends on tts-toolkit being actually usable"},
	{"elevenlabs", "ElevenLabs API", "set api_key in ~/.forge/config.toml [tts.elevenlabs]", "cloud backend; requires API key"},
	{"tts-toolkit", "tts-toolkit CLI wrapper", "pip3 install soundfile numpy torch", "repo path alone does not guarantee runtime health"},
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
				line := fmt.Sprintf("  %-14s  ✅ available (%s)", info.name, info.description)
				if info.configHint != "" {
					line += fmt.Sprintf(" — %s", info.configHint)
				}
				fmt.Println(line)
			} else {
				reason := info.installHint
				if setup := b.Setup(); setup != nil {
					reason = firstLine(strings.TrimSpace(setup.Error()))
				}
				fmt.Printf("  %-14s  ❌ unavailable (%s)\n", info.name, reason)
			}
		}

		fmt.Printf("\nDefault backend: %s\n", cfg.TTS.DefaultBackend)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(backendsCmd)
}
