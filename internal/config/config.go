package config

import (
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

type CorpusConfig struct {
	Paths []string `toml:"paths"`
}

type LLMConfig struct {
	Command string   `toml:"command"`
	Args    []string `toml:"args"`
}

type ProfileConfig struct {
	OutputDir string `toml:"output_dir"`
}

type TTSConfig struct {
	DefaultBackend string              `toml:"default_backend"`
	TTSToolkit     TTSToolkitConfig    `toml:"tts_toolkit"`
	ElevenLabs     ElevenLabsConfig    `toml:"elevenlabs"`
}

type TTSToolkitConfig struct {
	Path         string `toml:"path"`
	DefaultModel string `toml:"default_model"`
}

type ElevenLabsConfig struct {
	APIKey string `toml:"api_key"`
}

type VoicesConfig struct {
	Default string `toml:"default"`
}

type CharactersConfig struct {
	Dir     string `toml:"dir"`
	Default string `toml:"default"`
}

type Config struct {
	Corpus     CorpusConfig     `toml:"corpus"`
	LLM        LLMConfig        `toml:"llm"`
	Profile    ProfileConfig    `toml:"profile"`
	TTS        TTSConfig        `toml:"tts"`
	Voices     VoicesConfig     `toml:"voices"`
	Characters CharactersConfig `toml:"characters"`
}

func DefaultConfig() Config {
	return Config{
		Corpus: CorpusConfig{
			Paths: []string{
				"~/.openclaw/workspace/voice-corpus",
				"~/.openclaw/workspace/voice",
			},
		},
		LLM: LLMConfig{
			Command: "claude",
			Args:    []string{"--print", "--permission-mode", "bypassPermissions"},
		},
		Profile: ProfileConfig{
			OutputDir: "~/.forge/profile",
		},
		TTS: TTSConfig{
			DefaultBackend: "tts-toolkit",
			TTSToolkit: TTSToolkitConfig{
				Path:         "~/github/tts-toolkit",
				DefaultModel: "kokoro",
			},
		},
		Voices: VoicesConfig{
			Default: "cyperx",
		},
		Characters: CharactersConfig{
			Dir: "~/.forge/characters",
		},
	}
}

// ExpandPath replaces ~ with the user's home directory.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// ConfigPath returns the path to the config file.
func ConfigPath() string {
	return ExpandPath("~/.forge/config.toml")
}

// Load reads the config file or returns defaults if it doesn't exist.
func Load() (Config, error) {
	cfg := DefaultConfig()
	path := ConfigPath()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// Save writes the config to disk, creating directories as needed.
func Save(cfg Config) error {
	path := ConfigPath()
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := toml.Marshal(cfg)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// EnsureDefaults creates the default config if none exists.
func EnsureDefaults() error {
	path := ConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return Save(DefaultConfig())
	}
	return nil
}

// CorpusPaths returns expanded corpus paths from config.
func (c Config) CorpusPaths() []string {
	paths := make([]string, len(c.Corpus.Paths))
	for i, p := range c.Corpus.Paths {
		paths[i] = ExpandPath(p)
	}
	return paths
}

// ProfileDir returns the expanded profile output directory.
func (c Config) ProfileDir() string {
	return ExpandPath(c.Profile.OutputDir)
}

// CharactersDir returns the expanded characters directory.
func (c Config) CharactersDir() string {
	return ExpandPath(c.Characters.Dir)
}
