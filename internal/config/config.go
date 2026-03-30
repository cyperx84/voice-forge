package config

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

type CorpusConfig struct {
	Paths []string `toml:"paths"`
	Root  string   `toml:"root"`
	DB    string   `toml:"db"`
}

type LLMConfig struct {
	Command string   `toml:"command"`
	Args    []string `toml:"args"`
}

type ProfileConfig struct {
	OutputDir string `toml:"output_dir"`
}

type TTSConfig struct {
	DefaultBackend string           `toml:"default_backend"`
	TTSToolkit     TTSToolkitConfig `toml:"tts_toolkit"`
	ElevenLabs     ElevenLabsConfig `toml:"elevenlabs"`
	Chatterbox     ChatterboxConfig `toml:"chatterbox"`
	F5             F5Config         `toml:"f5"`
}

type ChatterboxConfig struct {
	VoicesDir   string `toml:"voices_dir"`
	RuntimePath string `toml:"runtime_path"`
}

type F5Config struct {
	VoicesDir   string `toml:"voices_dir"`
	RuntimePath string `toml:"runtime_path"`
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

type WatchConfig struct {
	Dir            string `toml:"dir"`
	Interval       string `toml:"interval"`
	FileWriteDelay string `toml:"file_write_delay"`
	WhisperCommand string `toml:"whisper_command"`
	WhisperModel   string `toml:"whisper_model"`
	OpenAIAPIKey   string `toml:"openai_api_key"`
}

type SkillConfig struct {
	Output     string `toml:"output"`
	AutoUpdate bool   `toml:"auto_update"`
}

type RefreshConfig struct {
	MinInterval       string `toml:"min_interval"`
	MinNewTranscripts int    `toml:"min_new_transcripts"`
}

type PreprocessConfig struct {
	SampleRate int     `toml:"sample_rate"`
	Channels   int     `toml:"channels"`
	BitDepth   int     `toml:"bit_depth"`
	Denoise    bool    `toml:"denoise"`
	MinSegment float64 `toml:"min_segment"`
	MaxSegment float64 `toml:"max_segment"`
}

type ScoringConfig struct {
	DefaultThreshold string `toml:"default_threshold"`
}

type ExportConfig struct {
	DefaultFormat string `toml:"default_format"`
	DefaultTier   string `toml:"default_tier"`
}

type EmbeddingConfig struct {
	Model string `toml:"model"`
}

type IngestConfig struct {
	AutoTag               bool     `toml:"auto_tag"`
	WhisperCommand        string   `toml:"whisper_command"`
	VideoKeyframeInterval int      `toml:"video_keyframe_interval"`
	DiscordExport         string   `toml:"discord_export"`
	TwitterArchive        string   `toml:"twitter_archive"`
	BulkVoiceDirs         []string `toml:"bulk_voice_dirs"`
	BulkCodeDirs          []string `toml:"bulk_code_dirs"`
	BulkTextDirs          []string `toml:"bulk_text_dirs"`
}

type FFmpegConfig struct {
	Threads int `toml:"threads"` // max threads for ffmpeg (0 = ffmpeg default)
	Nice    int `toml:"nice"`    // nice value on Unix (0 = no change)
}

type LiveDiscordConfig struct {
	Token         string `toml:"token"`
	VoiceChannel  string `toml:"voice_channel"`
	Guild         string `toml:"guild"`
	TargetUser    string `toml:"target_user"`
}

type LiveVADConfig struct {
	StartSensitivity  string `toml:"start_sensitivity"`
	EndSensitivity    string `toml:"end_sensitivity"`
	PrefixPaddingMs   int    `toml:"prefix_padding_ms"`
	SilenceDurationMs int    `toml:"silence_duration_ms"`
}

type LiveConfig struct {
	GeminiAPIKey   string           `toml:"gemini_api_key"`
	Model          string           `toml:"model"`
	Voice          string           `toml:"voice"`
	Language       string           `toml:"language"`
	Discord        LiveDiscordConfig `toml:"discord"`
	VAD            LiveVADConfig    `toml:"vad"`
	SystemPrompt   string           `toml:"system_prompt"`
	GreetingPrompt string           `toml:"greeting_prompt"`
}

type Config struct {
	Corpus     CorpusConfig     `toml:"corpus"`
	LLM        LLMConfig        `toml:"llm"`
	Profile    ProfileConfig    `toml:"profile"`
	TTS        TTSConfig        `toml:"tts"`
	Voices     VoicesConfig     `toml:"voices"`
	Characters CharactersConfig `toml:"characters"`
	Watch      WatchConfig      `toml:"watch"`
	Skill      SkillConfig      `toml:"skill"`
	Refresh    RefreshConfig    `toml:"refresh"`
	Preprocess PreprocessConfig `toml:"preprocess"`
	Scoring    ScoringConfig    `toml:"scoring"`
	Export     ExportConfig     `toml:"export"`
	Embedding  EmbeddingConfig  `toml:"embedding"`
	Ingest     IngestConfig     `toml:"ingest"`
	FFmpeg     FFmpegConfig     `toml:"ffmpeg"`
	Live       LiveConfig       `toml:"live"`
}

func DefaultConfig() Config {
	return Config{
		Corpus: CorpusConfig{
			Paths: []string{
				"~/.openclaw/workspace/voice-corpus",
				"~/.openclaw/workspace/voice",
			},
			Root: "~/.forge/corpus",
			DB:   "~/.forge/corpus.db",
		},
		LLM: LLMConfig{
			Command: "claude",
			Args:    []string{"--print", "--permission-mode", "bypassPermissions"},
		},
		Profile: ProfileConfig{
			OutputDir: "~/.forge/profile",
		},
		TTS: TTSConfig{
			DefaultBackend: "chatterbox",
			TTSToolkit: TTSToolkitConfig{
				Path:         "~/github/tts-toolkit",
				DefaultModel: "kokoro",
			},
			Chatterbox: ChatterboxConfig{
				VoicesDir:   "~/.forge/voices",
				RuntimePath: "~/.forge/venvs/chatterbox",
			},
			F5: F5Config{
				VoicesDir:   "~/.forge/voices",
				RuntimePath: "~/.forge/venvs/f5-tts",
			},
		},
		Voices: VoicesConfig{
			Default: "cyperx",
		},
		Characters: CharactersConfig{
			Dir: "~/.forge/characters",
		},
		Watch: WatchConfig{
			Dir:            "~/.openclaw/workspace/voice-corpus/",
			Interval:       "30s",
			FileWriteDelay: "500ms",
			WhisperCommand: "whisper-cli",
		},
		Skill: SkillConfig{
			Output:     "~/.openclaw/skills/cyperx-voice/",
			AutoUpdate: true,
		},
		Refresh: RefreshConfig{
			MinInterval:       "24h",
			MinNewTranscripts: 20,
		},
		Preprocess: PreprocessConfig{
			SampleRate: 24000,
			Channels:   1,
			BitDepth:   16,
			Denoise:    true,
			MinSegment: 3.0,
			MaxSegment: 15.0,
		},
		Scoring: ScoringConfig{
			DefaultThreshold: "silver",
		},
		Export: ExportConfig{
			DefaultFormat: "ljspeech",
			DefaultTier:   "silver",
		},
		Embedding: EmbeddingConfig{
			Model: "resemblyzer",
		},
		Ingest: IngestConfig{
			WhisperCommand:        "whisper-cli",
			VideoKeyframeInterval: 10,
		},
		FFmpeg: FFmpegConfig{
			Threads: 4,
			Nice:    10,
		},
		Live: LiveConfig{
			Model:    "gemini-3.1-flash-live-preview",
			Voice:    "Orus",
			Language: "en-US",
			VAD: LiveVADConfig{
				PrefixPaddingMs:   20,
				SilenceDurationMs: 100,
			},
		},
	}
}

// ExpandPath replaces ~ with the user's home directory.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			log.Printf("warning: cannot expand ~/ in path %q: %v", path, err)
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

	return os.WriteFile(path, data, 0600)
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

// WatchDir returns the expanded watch directory.
func (c Config) WatchDir() string {
	return ExpandPath(c.Watch.Dir)
}

// SkillOutputDir returns the expanded skill output directory.
func (c Config) SkillOutputDir() string {
	return ExpandPath(c.Skill.Output)
}

// CorpusRoot returns the expanded corpus root directory.
func (c Config) CorpusRoot() string {
	return ExpandPath(c.Corpus.Root)
}

// CorpusDBPath returns the expanded corpus database path.
func (c Config) CorpusDBPath() string {
	return ExpandPath(c.Corpus.DB)
}

// VoicesDir returns the expanded voices directory (for Chatterbox/F5 reference audio).
func (c Config) VoicesDir() string {
	if c.TTS.Chatterbox.VoicesDir != "" {
		return ExpandPath(c.TTS.Chatterbox.VoicesDir)
	}
	return ExpandPath("~/.forge/voices")
}
