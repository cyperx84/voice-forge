package cmd

import (
	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "forge",
	Short: "Voice corpus management, style extraction, and character voices",
	Long: `Voice Forge — personal voice corpus management, style extraction, and character voice CLI.

Reads voice message transcripts, extracts your unique speaking style,
and makes it programmable. Create characters with tone presets, generate
text in their voice, and speak through TTS backends.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return config.EnsureDefaults()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(liveCmd)
}
