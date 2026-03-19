package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/corpus"
	"github.com/cyperx84/voice-forge/internal/tts"
	"github.com/spf13/cobra"
)

var cloneBackend string
var cloneModel string
var cloneName string
var cloneMaxSamples int

var cloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "Clone your voice using corpus audio samples",
	Long: `Creates a voice clone using audio samples from your corpus.

Automatically selects the best samples (longest recordings) and sends
them to the specified TTS backend for voice cloning.

Examples:
  forge clone --backend tts-toolkit --model xtts
  forge clone --backend elevenlabs --name CyperX
  forge clone --name myvoice --max-samples 10`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		initBackends(cfg)

		backendName := cloneBackend
		if backendName == "" {
			backendName = cfg.TTS.DefaultBackend
		}
		if backendName == "" {
			return fmt.Errorf("no TTS backend configured — set [tts] default_backend in ~/.forge/config.toml or use --backend flag")
		}

		b, err := tts.Get(backendName)
		if err != nil {
			return err
		}

		if !b.Available() {
			return fmt.Errorf("backend %q is not available: %v", backendName, b.Setup())
		}

		name := cloneName
		if name == "" {
			name = cfg.Voices.Default
		}
		if name == "" {
			name = "cyperx"
		}

		// Find audio samples from corpus
		samples, err := findBestSamples(cfg, cloneMaxSamples)
		if err != nil {
			return fmt.Errorf("finding samples: %w", err)
		}
		if len(samples) == 0 {
			return fmt.Errorf("no audio samples found in corpus directories")
		}

		fmt.Printf("Cloning voice %q using %d samples via %s\n", name, len(samples), backendName)

		if err := b.Clone(samples, name); err != nil {
			return fmt.Errorf("voice cloning failed: %w", err)
		}

		// Create voices directory
		voiceDir := config.ExpandPath(fmt.Sprintf("~/.forge/voices/%s", name))
		if err := os.MkdirAll(voiceDir, 0755); err != nil {
			return fmt.Errorf("creating voice directory: %w", err)
		}

		// Write metadata using proper TOML marshaling
		meta := struct {
			Name    string `toml:"name"`
			Backend string `toml:"backend"`
			Samples int    `toml:"samples"`
		}{
			Name:    name,
			Backend: backendName,
			Samples: len(samples),
		}
		metaData, err := toml.Marshal(meta)
		if err != nil {
			return fmt.Errorf("marshaling voice metadata: %w", err)
		}
		metaPath := filepath.Join(voiceDir, "voice.toml")
		if err := os.WriteFile(metaPath, metaData, 0644); err != nil {
			return fmt.Errorf("writing voice metadata: %w", err)
		}

		fmt.Printf("Voice %q cloned successfully — saved to %s\n", name, voiceDir)
		return nil
	},
}

func init() {
	cloneCmd.Flags().StringVar(&cloneBackend, "backend", "", "TTS backend to use for cloning")
	cloneCmd.Flags().StringVar(&cloneModel, "model", "", "model to use for cloning (e.g. xtts)")
	cloneCmd.Flags().StringVar(&cloneName, "name", "", "name for the cloned voice")
	cloneCmd.Flags().IntVar(&cloneMaxSamples, "max-samples", 5, "maximum number of audio samples to use")
	rootCmd.AddCommand(cloneCmd)
}

// findBestSamples finds the longest audio recordings from the corpus.
func findBestSamples(cfg config.Config, maxSamples int) ([]string, error) {
	paths := cfg.CorpusPaths()

	// Look for audio files (wav, mp3, flac) in corpus dirs
	var audioFiles []string
	for _, dir := range paths {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			ext := filepath.Ext(e.Name())
			if ext == ".wav" || ext == ".mp3" || ext == ".flac" || ext == ".ogg" {
				audioFiles = append(audioFiles, filepath.Join(dir, e.Name()))
			}
		}
	}

	if len(audioFiles) == 0 {
		// Try using manifest to find recordings by UUID
		for _, dir := range paths {
			manifest := filepath.Join(dir, "manifest.txt")
			recordings, err := corpus.ParseManifest(manifest)
			if err != nil {
				continue
			}
			// Sort by duration (longest first)
			sort.Slice(recordings, func(i, j int) bool {
				return recordings[i].Duration > recordings[j].Duration
			})
			for _, rec := range recordings {
				for _, ext := range []string{".wav", ".mp3", ".flac", ".ogg"} {
					candidate := filepath.Join(dir, rec.UUID+ext)
					if _, err := os.Stat(candidate); err == nil {
						audioFiles = append(audioFiles, candidate)
						break
					}
				}
			}
		}
	}

	// Sort by file size (largest = longest recordings as a heuristic)
	sort.Slice(audioFiles, func(i, j int) bool {
		fi, _ := os.Stat(audioFiles[i])
		fj, _ := os.Stat(audioFiles[j])
		if fi == nil || fj == nil {
			return false
		}
		return fi.Size() > fj.Size()
	})

	if len(audioFiles) > maxSamples {
		audioFiles = audioFiles[:maxSamples]
	}

	return audioFiles, nil
}
