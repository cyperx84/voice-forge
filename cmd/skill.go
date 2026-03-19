package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/profile"
	"github.com/cyperx84/voice-forge/internal/skill"
	"github.com/spf13/cobra"
)

var skillCmd = &cobra.Command{
	Use:   "skill",
	Short: "Generate an OpenClaw agent skill from the style profile",
	Long: `Reads the style profile and generates OpenClaw skill files:
  - SKILL.md — instructions for agents to write in your voice
  - references/voice-profile.md — full style profile
  - references/avoid-list.md — words/phrases to never use
  - references/key-phrases.md — signature phrases and expressions

Examples:
  forge skill
  forge skill --output ~/.openclaw/skills/cyperx-voice/`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		output, _ := cmd.Flags().GetString("output")
		if output == "" {
			output = cfg.SkillOutputDir()
		} else {
			output = config.ExpandPath(output)
		}

		stylePath := filepath.Join(cfg.ProfileDir(), "style.json")
		p, err := profile.Load(stylePath)
		if err != nil {
			return fmt.Errorf("loading style profile: %w (run 'forge analyze' or 'forge refresh' first)", err)
		}

		if err := skill.Generate(p, output); err != nil {
			return fmt.Errorf("generating skill: %w", err)
		}

		fmt.Printf("OpenClaw skill generated at %s\n", output)
		fmt.Println("  SKILL.md")
		fmt.Println("  references/voice-profile.md")
		fmt.Println("  references/avoid-list.md")
		fmt.Println("  references/key-phrases.md")
		return nil
	},
}

func init() {
	skillCmd.Flags().String("output", "", "output directory for skill files (default from config)")
	rootCmd.AddCommand(skillCmd)
}
