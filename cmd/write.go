package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/cyperx84/voice-forge/internal/character"
	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/rewriter"
	"github.com/spf13/cobra"
)

var writeCharacter string

var writeCmd = &cobra.Command{
	Use:   "write [topic]",
	Short: "Generate text in a character's voice",
	Long: `Generate text on a topic using the style profile and a character's tone.

Uses the LLM to generate text matching the character's voice. Output goes to stdout.

Examples:
  forge write "why I switched to Go" --character podcast-host
  forge write "the future of AI" --character narrator
  forge write "launching my new project" --character hype`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		topic := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		charName := writeCharacter
		if charName == "" {
			charName = cfg.Characters.Default
		}
		if charName == "" {
			return fmt.Errorf("no character specified — use --character flag or set [characters] default in ~/.forge/config.toml")
		}

		ch, err := character.Get(charName, cfg.CharactersDir())
		if err != nil {
			return fmt.Errorf("loading character %q: %w", charName, err)
		}

		stylePath := filepath.Join(cfg.ProfileDir(), "style.json")
		styleJSON, err := rewriter.LoadStyleJSON(stylePath)
		if err != nil {
			// If no style profile exists, use empty JSON
			styleJSON = "{}"
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: no style profile found at %s — generating without base style\n", stylePath)
		}

		fmt.Fprintf(cmd.ErrOrStderr(), "Generating as %q...\n", ch.Name)
		text, err := rewriter.Generate(topic, ch, styleJSON, cfg.LLM.Command, cfg.LLM.Args)
		if err != nil {
			return fmt.Errorf("generating text: %w", err)
		}

		fmt.Println(text)
		return nil
	},
}

func init() {
	writeCmd.Flags().StringVar(&writeCharacter, "character", "", "character to write as")
	rootCmd.AddCommand(writeCmd)
}
