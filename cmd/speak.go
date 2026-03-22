package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cyperx84/voice-forge/internal/audioout"
	"github.com/cyperx84/voice-forge/internal/character"
	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/ffmpeg"
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
var speakPreset string
var speakNormalize bool
var speakListenLink bool
var speakListenTitle string

var speakCmd = &cobra.Command{
	Use:   "speak [text]",
	Short: "Generate speech audio from text",
	Long: `Converts text to speech using a configured TTS backend.

Uses the default backend from config, overridable with --backend.
Outputs audio to a file with --output, or writes bytes to stdout.

Use --preset to transcode the output to a specific format:
  discord   — 48kHz mono 128k MP3 (Discord attachment player)
  podcast   — 44.1kHz stereo 192k MP3 (long-form content)
  video     — 48kHz stereo AAC (ffmpeg-ready)
  lossless  — 44.1kHz mono 16-bit WAV (editing pipelines)

Use --normalize to convert output to standard mixing format (44.1kHz pcm_s16le mono WAV).
Use --listen-link to emit a self-contained HTML player page.

Examples:
  forge speak "let's rock and roll" --output test.wav
  forge speak "hello world" --backend elevenlabs --voice CyperX
  forge speak "ship it" --voice cyperx --preset discord --output ./out/cyperx.mp3
  forge speak "welcome to the show" --preset podcast --output episode.mp3 --listen-link`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		text := args[0]
		if strings.TrimSpace(text) == "" {
			return fmt.Errorf("text must not be empty")
		}
		const maxTextLen = 10000
		if len(text) > maxTextLen {
			return fmt.Errorf("text too long (%d chars, max %d) — split into smaller chunks", len(text), maxTextLen)
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
				fmt.Fprintf(os.Stderr, "Warning: could not load style profile %s: %v (using empty style)\n", stylePath, err)
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

		// Resolve preset: --discord is an alias for --preset discord
		presetName := speakPreset
		if speakDiscord && presetName == "" {
			presetName = "discord"
		}

		var preset *audioout.Preset
		if presetName != "" {
			p, ok := audioout.Presets[presetName]
			if !ok {
				return fmt.Errorf("unknown preset %q — available: %v", presetName, audioout.PresetNames())
			}
			preset = &p
		}

		ffCfg := ffmpeg.Config{Threads: cfg.FFmpeg.Threads, Nice: cfg.FFmpeg.Nice}

		// When transcoding with a preset, generate raw WAV first then transcode
		needsTranscode := preset != nil || speakNormalize
		format := speakFormat
		if needsTranscode {
			format = "wav"
		} else if format == "" {
			format = "wav"
		}

		backendOutput := finalOutput
		if needsTranscode {
			if finalOutput == "" {
				ext := "wav"
				if preset != nil {
					ext = preset.Format
				}
				tmp, err := os.CreateTemp("", "forge-out-*."+ext)
				if err != nil {
					return fmt.Errorf("creating temp output: %w", err)
				}
				finalOutput = tmp.Name()
				tmp.Close()
			}

			// Ensure correct file extension for preset
			if preset != nil {
				ext := strings.ToLower(filepath.Ext(finalOutput))
				wantExt := "." + preset.Format
				if ext == "" {
					finalOutput += wantExt
				} else if ext != wantExt {
					return fmt.Errorf("--preset %s expects %s output, got %s", preset.Name, wantExt, finalOutput)
				}
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

		if needsTranscode {
			if err := os.WriteFile(backendOutput, audio, 0644); err != nil {
				return fmt.Errorf("writing temp audio file: %w", err)
			}
			if preset != nil {
				if err := audioout.Transcode(backendOutput, finalOutput, *preset, ffCfg); err != nil {
					return err
				}
				fmt.Printf("%s audio written to %s\n", strings.Title(preset.Name), finalOutput)
			} else {
				// --normalize without preset
				if finalOutput == "" {
					tmp, err := os.CreateTemp("", "forge-norm-*.wav")
					if err != nil {
						return fmt.Errorf("creating temp output: %w", err)
					}
					finalOutput = tmp.Name()
					tmp.Close()
				}
				if err := audioout.Normalize(backendOutput, finalOutput, ffCfg); err != nil {
					return err
				}
				fmt.Printf("Normalized audio written to %s\n", finalOutput)
			}

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
				return fmt.Errorf("--listen-link requires --output or --preset so there is a file to wrap")
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
	speakCmd.Flags().StringVar(&speakPreset, "preset", "", "output preset (discord, podcast, video, lossless)")
	speakCmd.Flags().BoolVar(&speakNormalize, "normalize", false, "normalize output to standard mixing format (44.1kHz pcm_s16le mono WAV)")
	speakCmd.Flags().BoolVar(&speakDiscord, "discord", false, "alias for --preset discord")
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
