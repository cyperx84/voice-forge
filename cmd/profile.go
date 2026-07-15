package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/cyperx84/voice-forge/internal/config"
	prof "github.com/cyperx84/voice-forge/internal/profile"
	"github.com/spf13/cobra"
)

var (
	briefFlag         bool
	profileOutputFlag string
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Display your current voice style profile",
	Long: `Reads and pretty-prints the style profile from ~/.forge/profile/style.json.
Use --brief for a quick summary or omit for the full dump.

Examples:
  forge profile
  forge profile --brief
  forge profile --output ./voice-profile.json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		profilePath := filepath.Join(cfg.ProfileDir(), "style.json")
		profile, err := prof.Load(profilePath)
		if err != nil {
			return fmt.Errorf("no style profile found — run 'forge analyze' first\n  expected: %s", profilePath)
		}

		if profileOutputFlag != "" {
			outputPath := config.ExpandPath(profileOutputFlag)
			if err := prof.ExportPortable(profile, outputPath); err != nil {
				return fmt.Errorf("export profile: %w", err)
			}
			fmt.Printf("Exported portable voice profile to %s\n", outputPath)
			return nil
		}

		if briefFlag {
			prof.PrintBrief(profile)
		} else {
			prof.PrintFull(profile)
		}

		return nil
	},
}

func init() {
	profileCmd.Flags().BoolVar(&briefFlag, "brief", false, "Show brief summary only")
	profileCmd.Flags().StringVarP(&profileOutputFlag, "output", "o", "", "Export a portable, versioned JSON profile")
	rootCmd.AddCommand(profileCmd)
}
