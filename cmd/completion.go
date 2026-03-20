package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate completion scripts for your shell.

To load completions:

Bash:
  $ source <(forge completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ forge completion bash > /etc/bash_completion.d/forge
  # macOS:
  $ forge completion bash > $(brew --prefix)/etc/bash_completion.d/forge

Zsh:
  $ source <(forge completion zsh)
  # To load completions for each session, execute once:
  $ forge completion zsh > "${fpath[1]}/_forge"

Fish:
  $ forge completion fish | source
  # To load completions for each session, execute once:
  $ forge completion fish > ~/.config/fish/completions/forge.fish

PowerShell:
  PS> forge completion powershell | Out-String | Invoke-Expression
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		switch args[0] {
		case "bash":
			return rootCmd.GenBashCompletion(os.Stdout)
		case "zsh":
			return rootCmd.GenZshCompletion(os.Stdout)
		case "fish":
			return rootCmd.GenFishCompletion(os.Stdout, true)
		case "powershell":
			return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}
