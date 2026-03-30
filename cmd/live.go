package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/cyperx84/voice-forge/internal/character"
	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/live"
	"github.com/spf13/cobra"
)

// ── forge live ────────────────────────────────────────────────────────────────

var liveCmd = &cobra.Command{
	Use:   "live",
	Short: "Manage Gemini Live voice sessions",
	Long:  `Start, stop, and configure real-time Gemini Live voice sessions (Discord or local).`,
}

// ── forge live start ──────────────────────────────────────────────────────────

var liveStartCharacter string
var liveStartVoice string
var liveStartLang string
var liveStartPrompt string

var liveStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start a live session",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		liveCfg := cfg.Live

		// Override voice / language from flags.
		if liveStartVoice != "" {
			liveCfg.Voice = liveStartVoice
		}
		if liveStartLang != "" {
			liveCfg.Language = liveStartLang
		}

		// Build system prompt from character or explicit flag.
		if liveStartPrompt != "" {
			liveCfg.SystemPrompt = liveStartPrompt
		} else if liveStartCharacter != "" {
			ch, err := character.Get(liveStartCharacter, cfg.CharactersDir())
			if err != nil {
				return fmt.Errorf("loading character %q: %w", liveStartCharacter, err)
			}
			stylePath := filepath.Join(cfg.ProfileDir(), "style.json")
			liveCfg.SystemPrompt = live.CharacterToSystemPrompt(*ch, stylePath)
			fmt.Fprintf(cmd.ErrOrStderr(), "Using character %q → system prompt set.\n", ch.Name)
		}

		if err := live.Start(liveCfg); err != nil {
			return err
		}

		fmt.Println("Live session started.")
		return nil
	},
}

// ── forge live stop ───────────────────────────────────────────────────────────

var liveStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the running live session",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := live.Stop(); err != nil {
			return err
		}
		fmt.Println("Live session stopped.")
		return nil
	},
}

// ── forge live status ─────────────────────────────────────────────────────────

var liveStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show live session status",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := live.Status()
		if !s.Running {
			fmt.Println("Status: stopped")
			return nil
		}
		fmt.Println("Status:  running")
		fmt.Printf("PID:     %d\n", s.PID)
		fmt.Printf("Uptime:  %s\n", s.Uptime.Round(1e9))
		return nil
	},
}

// ── forge live voice ──────────────────────────────────────────────────────────

var liveVoiceCmd = &cobra.Command{
	Use:   "voice [name]",
	Short: "List or set the Gemini Live voice",
	Long: `Without an argument, shows the current configured voice.
With an argument, updates [live] voice in ~/.forge/config.toml.

Available Gemini voices (30 total, e.g.):
  Orus, Puck, Gacrux, Leda, Algenib, Alya, Aoede, Charon, Despina,
  Enceladus, Erinome, Fenrir, Kore, Nyx, Oberon, Sulafat, Umbriel, Zephyr`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		if len(args) == 0 {
			fmt.Printf("Current voice: %s\n", cfg.Live.Voice)
			return nil
		}

		cfg.Live.Voice = args[0]
		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}
		fmt.Printf("Live voice set to %q.\n", args[0])
		return nil
	},
}

// ── forge live config ─────────────────────────────────────────────────────────

var liveConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Show live configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		lc := cfg.Live
		apiKeyDisplay := "(not set)"
		if lc.GeminiAPIKey != "" {
			apiKeyDisplay = "***" + lc.GeminiAPIKey[max(0, len(lc.GeminiAPIKey)-4):]
		}
		discordTokenDisplay := "(not set)"
		if lc.Discord.Token != "" {
			discordTokenDisplay = "(set)"
		}

		fmt.Printf("Model:              %s\n", lc.Model)
		fmt.Printf("Voice:              %s\n", lc.Voice)
		fmt.Printf("Language:           %s\n", lc.Language)
		fmt.Printf("Gemini API key:     %s\n", apiKeyDisplay)
		fmt.Printf("System prompt:      %s\n", truncate(lc.SystemPrompt, 60))
		fmt.Printf("Greeting prompt:    %s\n", truncate(lc.GreetingPrompt, 60))
		fmt.Println()
		fmt.Println("[discord]")
		fmt.Printf("  Token:         %s\n", discordTokenDisplay)
		fmt.Printf("  Voice channel: %s\n", lc.Discord.VoiceChannel)
		fmt.Printf("  Guild:         %s\n", lc.Discord.Guild)
		fmt.Printf("  Target user:   %s\n", lc.Discord.TargetUser)
		fmt.Println()
		fmt.Println("[vad]")
		fmt.Printf("  Start sensitivity:  %s\n", lc.VAD.StartSensitivity)
		fmt.Printf("  End sensitivity:    %s\n", lc.VAD.EndSensitivity)
		fmt.Printf("  Prefix padding ms:  %d\n", lc.VAD.PrefixPaddingMs)
		fmt.Printf("  Silence duration ms:%d\n", lc.VAD.SilenceDurationMs)
		fmt.Println()
		fmt.Printf("Bot directory: %s\n", live.BotDir())

		return nil
	},
}

// ── helpers ───────────────────────────────────────────────────────────────────

func truncate(s string, n int) string {
	if s == "" {
		return "(not set)"
	}
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ── init ──────────────────────────────────────────────────────────────────────

func init() {
	liveStartCmd.Flags().StringVar(&liveStartCharacter, "character", "", "character to embody (sets system prompt)")
	liveStartCmd.Flags().StringVar(&liveStartVoice, "voice", "", "Gemini voice name")
	liveStartCmd.Flags().StringVar(&liveStartLang, "lang", "", "language code (e.g. en-US)")
	liveStartCmd.Flags().StringVar(&liveStartPrompt, "prompt", "", "system prompt (overrides --character)")

	liveCmd.AddCommand(liveStartCmd)
	liveCmd.AddCommand(liveStopCmd)
	liveCmd.AddCommand(liveStatusCmd)
	liveCmd.AddCommand(liveVoiceCmd)
	liveCmd.AddCommand(liveConfigCmd)
}
