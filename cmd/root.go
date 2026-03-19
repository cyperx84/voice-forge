package cmd

import (
	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "forge",
	Short: "Voice corpus management and style extraction",
	Long: `Voice Forge — personal voice corpus management and style extraction CLI.

Reads voice message transcripts, extracts your unique speaking style,
and makes it programmable. Feed it your corpus, get back your voice DNA.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return config.EnsureDefaults()
	},
}

func Execute() error {
	return rootCmd.Execute()
}
