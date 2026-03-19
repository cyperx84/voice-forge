package cmd

import (
	"fmt"
	"os"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/tts"
	"github.com/spf13/cobra"
)

var speakBackend string
var speakVoice string
var speakOutput string
var speakSpeed float64
var speakFormat string

var speakCmd = &cobra.Command{
	Use:   "speak [text]",
	Short: "Generate speech audio from text",
	Long: `Converts text to speech using a configured TTS backend.

Uses the default backend from config, overridable with --backend.
Outputs audio to a file with --output, or prints the path of a temp file.

Examples:
  forge speak "let's rock and roll" --output test.wav
  forge speak "hello world" --backend elevenlabs --voice CyperX
  forge speak "testing" --backend tts-toolkit --voice kokoro`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		text := args[0]

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
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

		voice := speakVoice
		if voice == "" {
			voice = cfg.Voices.Default
		}

		format := speakFormat
		if format == "" {
			format = "wav"
		}

		opts := tts.SpeakOpts{
			Voice:      voice,
			Speed:      speakSpeed,
			OutputPath: speakOutput,
			Format:     format,
		}

		audio, err := b.Speak(text, opts)
		if err != nil {
			return fmt.Errorf("speech generation failed: %w", err)
		}

		if speakOutput != "" {
			if err := os.WriteFile(speakOutput, audio, 0644); err != nil {
				return fmt.Errorf("writing output file: %w", err)
			}
			fmt.Printf("Audio written to %s (%d bytes)\n", speakOutput, len(audio))
		} else {
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
	rootCmd.AddCommand(speakCmd)
}

func initBackends(cfg config.Config) {
	toolkitPath := cfg.TTS.TTSToolkit.Path
	if toolkitPath == "" {
		toolkitPath = config.ExpandPath("~/github/tts-toolkit")
	}

	toolkit := &tts.ToolkitBackend{
		Path:         config.ExpandPath(toolkitPath),
		DefaultModel: cfg.TTS.TTSToolkit.DefaultModel,
	}
	tts.Register(toolkit)

	apiKey := cfg.TTS.ElevenLabs.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ELEVENLABS_API_KEY")
	}
	tts.Register(&tts.ElevenLabsBackend{APIKey: apiKey})

	tts.Register(&tts.KokoroBackend{Toolkit: toolkit})
}
