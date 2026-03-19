package cmd

import (
	"fmt"

	"github.com/cyperx84/voice-forge/internal/character"
	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/spf13/cobra"
)

var characterCmd = &cobra.Command{
	Use:   "character",
	Short: "Manage voice characters",
	Long:  `Characters are style mutations on top of your base voice DNA. Each character applies tone shifts, pacing changes, and vocabulary adjustments to create a distinct voice persona.`,
}

var characterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all characters",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		chars, err := character.List(cfg.CharactersDir())
		if err != nil {
			return fmt.Errorf("listing characters: %w", err)
		}

		if len(chars) == 0 {
			fmt.Println("No characters found.")
			return nil
		}

		fmt.Printf("%-15s %-15s %s\n", "NAME", "BASE", "DESCRIPTION")
		fmt.Printf("%-15s %-15s %s\n", "----", "----", "-----------")
		for _, ch := range chars {
			base := ch.BasedOn
			if base == "" {
				base = "-"
			}
			fmt.Printf("%-15s %-15s %s\n", ch.Name, base, ch.Description)
		}

		return nil
	},
}

var characterShowCmd = &cobra.Command{
	Use:   "show [name]",
	Short: "Display character details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		ch, err := character.Get(args[0], cfg.CharactersDir())
		if err != nil {
			return fmt.Errorf("loading character: %w", err)
		}

		fmt.Printf("Name:        %s\n", ch.Name)
		fmt.Printf("Description: %s\n", ch.Description)
		fmt.Printf("Based on:    %s\n", ch.BasedOn)
		fmt.Println()
		fmt.Println("Tone Shift:")
		fmt.Printf("  Register:    %s\n", ch.ToneShift.Register)
		fmt.Printf("  Pacing:      %s\n", ch.ToneShift.Pacing)
		fmt.Printf("  Emoji style: %s\n", ch.ToneShift.EmojiStyle)
		fmt.Printf("  Persona:     %s\n", ch.ToneShift.Persona)

		if len(ch.ToneShift.Vocabulary) > 0 {
			fmt.Println("  Vocabulary:")
			for _, w := range ch.ToneShift.Vocabulary {
				fmt.Printf("    + %s\n", w)
			}
		}
		if len(ch.ToneShift.AvoidWords) > 0 {
			fmt.Println("  Avoid words:")
			for _, w := range ch.ToneShift.AvoidWords {
				fmt.Printf("    - %s\n", w)
			}
		}

		if len(ch.VoiceOpts) > 0 {
			fmt.Println()
			fmt.Println("Voice options:")
			for k, v := range ch.VoiceOpts {
				fmt.Printf("  %s: %s\n", k, v)
			}
		}

		return nil
	},
}

var (
	createBase     string
	createRegister string
	createPacing   string
	createPersona  string
	createEmoji    string
	createDesc     string
)

var characterCreateCmd = &cobra.Command{
	Use:   "create [name]",
	Short: "Create a new character",
	Long: `Create a new voice character with tone shift settings.

Examples:
  forge character create narrator --base cyperx --register formal --pacing slow --persona "Documentary narrator"
  forge character create hype-bot --register ultra-casual --pacing fast --description "Maximum energy mode"`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		ch := &character.Character{
			Name:        name,
			Description: createDesc,
			BasedOn:     createBase,
			ToneShift: character.ToneShift{
				Register:   createRegister,
				Pacing:     createPacing,
				Persona:    createPersona,
				EmojiStyle: createEmoji,
			},
		}

		dir := cfg.CharactersDir()
		if err := character.Save(ch, dir); err != nil {
			return fmt.Errorf("saving character: %w", err)
		}

		fmt.Printf("Character %q created at %s/%s.toml\n", name, dir, name)
		return nil
	},
}

func init() {
	characterCreateCmd.Flags().StringVar(&createBase, "base", "cyperx", "base voice to build on")
	characterCreateCmd.Flags().StringVar(&createRegister, "register", "casual", "tone register (formal, casual, dramatic, etc.)")
	characterCreateCmd.Flags().StringVar(&createPacing, "pacing", "variable", "pacing (slow, measured, fast, breathless)")
	characterCreateCmd.Flags().StringVar(&createPersona, "persona", "", "free-form persona description")
	characterCreateCmd.Flags().StringVar(&createEmoji, "emoji", "none", "emoji style (none, minimal, heavy)")
	characterCreateCmd.Flags().StringVar(&createDesc, "description", "", "character description")

	characterCmd.AddCommand(characterListCmd)
	characterCmd.AddCommand(characterShowCmd)
	characterCmd.AddCommand(characterCreateCmd)
	rootCmd.AddCommand(characterCmd)
}
