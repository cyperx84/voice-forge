package live

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cyperx84/voice-forge/internal/character"
	"github.com/cyperx84/voice-forge/internal/config"
)

// styleProfile is a minimal representation of ~/.forge/profile/style.json.
type styleProfile struct {
	Persona    string   `json:"persona"`
	Register   string   `json:"register"`
	Pacing     string   `json:"pacing"`
	Vocabulary []string `json:"vocabulary"`
	AvoidWords []string `json:"avoid_words"`
}

func loadBaseProfile(path string) (styleProfile, error) {
	var sp styleProfile
	data, err := os.ReadFile(path)
	if err != nil {
		return sp, err
	}
	if err := json.Unmarshal(data, &sp); err != nil {
		return sp, fmt.Errorf("parsing style.json: %w", err)
	}
	return sp, nil
}

// CharacterToSystemPrompt converts a Character into a Gemini Live system prompt.
// If baseProfile is empty it attempts to load ~/.forge/profile/style.json.
func CharacterToSystemPrompt(ch character.Character, baseProfile string) string {
	var base styleProfile

	profilePath := baseProfile
	if profilePath == "" {
		profilePath = config.ExpandPath("~/.forge/profile/style.json")
	}
	if p, err := loadBaseProfile(profilePath); err == nil {
		base = p
	}

	var sb strings.Builder

	// Persona line
	persona := ch.ToneShift.Persona
	if persona == "" && ch.Description != "" {
		persona = ch.Description
	}
	if persona == "" && base.Persona != "" {
		persona = base.Persona
	}
	if persona != "" {
		fmt.Fprintf(&sb, "You are %s.", persona)
	} else if ch.Name != "" {
		fmt.Fprintf(&sb, "You are %s.", ch.Name)
	}

	// Style line
	register := ch.ToneShift.Register
	if register == "" {
		register = base.Register
	}
	pacing := ch.ToneShift.Pacing
	if pacing == "" {
		pacing = base.Pacing
	}
	if register != "" || pacing != "" {
		sb.WriteString(" Style:")
		if register != "" {
			fmt.Fprintf(&sb, " %s register,", register)
		}
		if pacing != "" {
			fmt.Fprintf(&sb, " %s pace.", pacing)
		} else {
			// trim trailing comma if no pacing
			s := sb.String()
			sb.Reset()
			sb.WriteString(strings.TrimRight(s, ","))
			sb.WriteString(".")
		}
	}

	// Vocabulary
	vocab := ch.ToneShift.Vocabulary
	if len(vocab) == 0 {
		vocab = base.Vocabulary
	}
	if len(vocab) > 0 {
		fmt.Fprintf(&sb, " Use phrases like: %s.", strings.Join(vocab, "; "))
	}

	// Avoid words
	avoid := ch.ToneShift.AvoidWords
	if len(avoid) == 0 {
		avoid = base.AvoidWords
	}
	if len(avoid) > 0 {
		fmt.Fprintf(&sb, " Avoid: %s.", strings.Join(avoid, ", "))
	}

	// Emoji style
	if ch.ToneShift.EmojiStyle != "" && ch.ToneShift.EmojiStyle != "none" {
		fmt.Fprintf(&sb, " Emoji usage: %s.", ch.ToneShift.EmojiStyle)
	}

	return strings.TrimSpace(sb.String())
}
