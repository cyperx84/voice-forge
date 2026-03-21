package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cyperx84/voice-forge/internal/audioout"
	"github.com/cyperx84/voice-forge/internal/character"
	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/rewriter"
	"github.com/cyperx84/voice-forge/internal/tts"
	"github.com/spf13/cobra"
)

var speakBackend string
var speakVoice string
var speakOutput string
var speakSpeed float64
var speakFormat string
var speakCharacter string
var speakDiscord bool
var speakListenLink bool
var speakListenTitle string

var speakCmd = &cobra.Command{
	Use:   "speak [text]",
	Short: "Generate speech audio from text",
	Long: `Converts text to speech using a configured TTS backend.

Uses the default backend from config, overridable with --backend.
Outputs audio to a file with --output, or writes bytes to stdout.

Use --discord to normalize the final file to a Discord-friendly MP3
attachment. Use --listen-link to emit a self-contained HTML player page
next to the final audio file.

Examples:
  forge speak "let's rock and roll" --output test.wav
  forge speak "hello world" --backend elevenlabs --voice CyperX
  forge speak "testing" --backend tts-toolkit --voice kokoro
  forge speak "ship it" --voice cyperx --discord --output ./out/cyperx.mp3 --listen-link`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		text := args[0]
		if strings.TrimSpace(text) == "" {
			return fmt.Errorf("text must not be empty")
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		// Apply character tone shift if specified
		charName := speakCharacter
		if charName == "" {
			charName = cfg.Characters.Default
		}
		if charName != "" {
			ch, err := character.Get(charName, cfg.CharactersDir())
			if err != nil {
				return fmt.Errorf("loading character %q: %w", charName, err)
			}

			stylePath := filepath.Join(cfg.ProfileDir(), "style.json")
			styleJSON, err := rewriter.LoadStyleJSON(stylePath)
			if err != nil {
				styleJSON = "{}"
			}

			fmt.Fprintf(os.Stderr, "Rewriting as %q...\n", ch.Name)
			rewritten, err := rewriter.Rewrite(text, ch, styleJSON, cfg.LLM.Command, cfg.LLM.Args)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: rewrite failed, using original text: %v\n", err)
			} else {
				text = rewritten
			}

			// Apply character voice opts to override voice selection
			if v, ok := ch.VoiceOpts["voice"]; ok && speakVoice == "" {
				speakVoice = v
			}
			if v, ok := ch.VoiceOpts["backend"]; ok && speakBackend == "" {
				speakBackend = v
			}
		}

		initBackends(cfg)

		backendName := speakBackend
		if backendName == "" {
			backendName = cfg.TTS.DefaultBackend
		}
		if backendName == "" {
			return fmt.Errorf("no TTS backend configured — set [tts] default_backend in ~/.forge/config.toml or use --backend flag\navailable backends: %v", tts.Names())
		}

		b, err := tts.Get(backendName)
		if err != nil {
			return err
		}

		if !b.Available() {
			return fmt.Errorf("backend %q is not available — run 'forge speak --backend %s' after configuring it\n%v", backendName, backendName, b.Setup())
		}

		finalOutput := speakOutput
		if finalOutput != "" {
			outDir := filepath.Dir(finalOutput)
			if info, err := os.Stat(outDir); err != nil || !info.IsDir() {
				return fmt.Errorf("output directory does not exist: %s", outDir)
			}
		}

		voice := speakVoice
		if voice == "" {
			voice = cfg.Voices.Default
		}

		format := speakFormat
		if speakDiscord {
			format = "wav"
		} else if format == "" {
			format = "wav"
		}

		backendOutput := finalOutput
		if speakDiscord {
			if finalOutput == "" {
				tmp, err := os.CreateTemp("", "forge-discord-*.mp3")
				if err != nil {
					return fmt.Errorf("creating temp output: %w", err)
				}
				finalOutput = tmp.Name()
				tmp.Close()
			}
			switch ext := strings.ToLower(filepath.Ext(finalOutput)); ext {
			case "":
				finalOutput += ".mp3"
			case ".mp3":
			default:
				return fmt.Errorf("--discord output must end in .mp3, got %s", finalOutput)
			}

			tmp, err := os.CreateTemp("", "forge-raw-*.wav")
			if err != nil {
				return fmt.Errorf("creating temp backend output: %w", err)
			}
			backendOutput = tmp.Name()
			tmp.Close()
			defer os.Remove(backendOutput)
		}

		opts := tts.SpeakOpts{
			Voice:      voice,
			Speed:      speakSpeed,
			OutputPath: backendOutput,
			Format:     format,
		}

		audio, err := b.Speak(text, opts)
		if err != nil {
			return fmt.Errorf("speech generation failed: %w", err)
		}

		if speakDiscord {
			if err := os.WriteFile(backendOutput, audio, 0644); err != nil {
				return fmt.Errorf("writing temp audio file: %w", err)
			}
			if err := audioout.TranscodeDiscordMP3(backendOutput, finalOutput); err != nil {
				return err
			}
			fmt.Printf("Discord-ready audio written to %s\n", finalOutput)

			if speakListenLink {
				pagePath, err := audioout.WriteListenPage(finalOutput, speakListenTitle, text)
				if err != nil {
					return err
				}
				fmt.Printf("Listen page written to %s\n", pagePath)
			}
			return nil
		}

		if finalOutput != "" {
			if err := os.WriteFile(finalOutput, audio, 0644); err != nil {
				return fmt.Errorf("writing output file: %w", err)
			}
			fmt.Printf("Audio written to %s (%d bytes)\n", finalOutput, len(audio))
			if speakListenLink {
				pagePath, err := audioout.WriteListenPage(finalOutput, speakListenTitle, text)
				if err != nil {
					return err
				}
				fmt.Printf("Listen page written to %s\n", pagePath)
			}
		} else {
			if speakListenLink {
				return fmt.Errorf("--listen-link requires --output or --discord so there is a file to wrap")
			}
			// Write to stdout for piping
			if _, err := os.Stdout.Write(audio); err != nil {
				return fmt.Errorf("writing to stdout: %w", err)
			}
		}

		return nil
	},
}

func init() {
	speakCmd.Flags().StringVar(&speakBackend, "backend", "", "TTS backend to use")
	speakCmd.Flags().StringVar(&speakVoice, "voice", "", "voice/model name")
	speakCmd.Flags().StringVarP(&speakOutput, "output", "o", "", "output file path")
	speakCmd.Flags().Float64Var(&speakSpeed, "speed", 0, "speech rate multiplier")
	speakCmd.Flags().StringVar(&speakFormat, "format", "", "output format (wav or mp3)")
	speakCmd.Flags().StringVar(&speakCharacter, "character", "", "character to speak as")
	speakCmd.Flags().BoolVar(&speakDiscord, "discord", false, "transcode final output to a Discord-friendly MP3 attachment via ffmpeg")
	speakCmd.Flags().BoolVar(&speakListenLink, "listen-link", false, "write a self-contained HTML player page next to the final audio file")
	speakCmd.Flags().StringVar(&speakListenTitle, "listen-title", "", "title to show in the generated listen page")
	rootCmd.AddCommand(speakCmd)
}

func initBackends(cfg config.Config) {
	voicesDir := cfg.VoicesDir()

	// Chatterbox (top priority)
	tts.Register(&tts.ChatterboxBackend{VoicesDir: voicesDir})

	// F5-TTS
	tts.Register(&tts.F5Backend{VoicesDir: voicesDir})

	// Toolkit
	toolkitPath := cfg.TTS.TTSToolkit.Path
	if toolkitPath == "" {
		toolkitPath = config.ExpandPath("~/github/tts-toolkit")
	}
	toolkit := &tts.ToolkitBackend{
		Path:         config.ExpandPath(toolkitPath),
		DefaultModel: cfg.TTS.TTSToolkit.DefaultModel,
	}
	tts.Register(toolkit)

	// Kokoro (via toolkit)
	tts.Register(&tts.KokoroBackend{Toolkit: toolkit})

	// ElevenLabs
	apiKey := cfg.TTS.ElevenLabs.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ELEVENLABS_API_KEY")
	}
	tts.Register(&tts.ElevenLabsBackend{APIKey: apiKey})
}
