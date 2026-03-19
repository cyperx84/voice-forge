package character

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

// Character represents a style mutation on top of the base voice DNA.
type Character struct {
	Name        string            `json:"name" toml:"name"`
	Description string            `json:"description" toml:"description"`
	BasedOn     string            `json:"based_on" toml:"based_on"`
	ToneShift   ToneShift         `json:"tone_shift" toml:"tone_shift"`
	VoiceOpts   map[string]string `json:"voice_opts" toml:"voice_opts"`
}

// ToneShift defines how the character's tone differs from the base voice.
type ToneShift struct {
	Register   string   `json:"register" toml:"register"`
	Pacing     string   `json:"pacing" toml:"pacing"`
	Vocabulary []string `json:"add_words" toml:"add_words"`
	AvoidWords []string `json:"avoid_words" toml:"avoid_words"`
	Persona    string   `json:"persona" toml:"persona"`
	EmojiStyle string   `json:"emoji_style" toml:"emoji_style"`
}

// builtinPresets contains the default character presets embedded in the binary.
var builtinPresets = []Character{
	{
		Name:        "narrator",
		Description: "Documentary narrator — measured, authoritative, deliberate",
		BasedOn:     "cyperx",
		ToneShift: ToneShift{
			Register:   "formal",
			Pacing:     "slow",
			Vocabulary: []string{"What followed was...", "The result was clear...", "In the end,", "As it turned out,"},
			AvoidWords: []string{"yo", "dude", "lol", "nah", "gonna", "wanna"},
			Persona:    "Documentary narrator: measured and authoritative. Drops slang, keeps technical terms. Uses deliberate pacing with smooth transitions between ideas. Think David Attenborough meets tech journalism.",
			EmojiStyle: "none",
		},
	},
	{
		Name:        "podcast-host",
		Description: "Conversational podcast host — casual-professional, energetic",
		BasedOn:     "cyperx",
		ToneShift: ToneShift{
			Register:   "casual-professional",
			Pacing:     "variable",
			Vocabulary: []string{"So here's the thing...", "Let me break this down...", "And honestly?", "Here's what's wild about this..."},
			AvoidWords: []string{"per se", "aforementioned", "henceforth", "whereby"},
			Persona:    "Podcast host: keeps the enthusiasm and energy but adds structure and flow. Conversational but prepared. Asks rhetorical questions. Builds toward punchlines and key takeaways.",
			EmojiStyle: "minimal",
		},
	},
	{
		Name:        "storyteller",
		Description: "Engaging storyteller — dramatic-casual, builds tension",
		BasedOn:     "cyperx",
		ToneShift: ToneShift{
			Register:   "dramatic-casual",
			Pacing:     "variable",
			Vocabulary: []string{"Picture this:", "And then—", "But here's the twist:", "You could feel it in the air."},
			AvoidWords: []string{"basically", "essentially", "in conclusion"},
			Persona:    "Storyteller: uses sensory language and dramatic pacing. Builds tension, pauses for effect. Vivid descriptions, strong verbs. Draws the listener in with scene-setting before delivering the point.",
			EmojiStyle: "minimal",
		},
	},
	{
		Name:        "hype",
		Description: "Maximum energy — short punchy sentences, heavy on action",
		BasedOn:     "cyperx",
		ToneShift: ToneShift{
			Register:   "ultra-casual",
			Pacing:     "fast",
			Vocabulary: []string{"Let's go!", "Ship it!", "This is insane.", "No cap.", "Absolute game changer."},
			AvoidWords: []string{"however", "nevertheless", "furthermore", "in my opinion"},
			Persona:    "Maximum energy hype mode: turns enthusiasm up to 11. Short punchy sentences. Heavy on action words. Exclamation points welcome. Think product launch meets pep rally. Every sentence hits hard.",
			EmojiStyle: "heavy",
		},
	},
}

// Load reads a character from a TOML file.
func Load(path string) (*Character, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading character file: %w", err)
	}

	var ch Character
	if err := toml.Unmarshal(data, &ch); err != nil {
		return nil, fmt.Errorf("parsing character file: %w", err)
	}

	return &ch, nil
}

// Save writes a character to a TOML file.
func Save(ch *Character, dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating characters directory: %w", err)
	}

	data, err := toml.Marshal(ch)
	if err != nil {
		return fmt.Errorf("marshaling character: %w", err)
	}

	path := filepath.Join(dir, ch.Name+".toml")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing character file: %w", err)
	}

	return nil
}

// EnsurePresets writes built-in presets to the characters directory if they don't exist.
func EnsurePresets(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating characters directory: %w", err)
	}

	for _, preset := range builtinPresets {
		path := filepath.Join(dir, preset.Name+".toml")
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := Save(&preset, dir); err != nil {
				return err
			}
		}
	}

	return nil
}

// List returns all characters from the characters directory.
func List(dir string) ([]Character, error) {
	if err := EnsurePresets(dir); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading characters directory: %w", err)
	}

	var chars []Character
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}
		ch, err := Load(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		chars = append(chars, *ch)
	}

	return chars, nil
}

// Get loads a character by name from the characters directory.
func Get(name, dir string) (*Character, error) {
	if err := EnsurePresets(dir); err != nil {
		return nil, err
	}

	path := filepath.Join(dir, name+".toml")
	return Load(path)
}
