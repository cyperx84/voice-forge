package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cyperx84/voice-forge/internal/analyzer"
)

// Load reads and parses the style.json profile.
func Load(path string) (*analyzer.StyleProfile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var profile analyzer.StyleProfile
	if err := json.Unmarshal(data, &profile); err != nil {
		return nil, err
	}

	return &profile, nil
}

// PrintBrief displays a short summary of the profile.
func PrintBrief(p *analyzer.StyleProfile) {
	fmt.Println("Voice Style Profile")
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("Analyzed:  %s\n", formatTime(p.AnalyzedAt))
	fmt.Printf("Samples:   %d\n", p.SampleCount)
	fmt.Printf("Words:     %d\n", p.TotalWords)
	fmt.Println()
	fmt.Println("Voice DNA:")
	fmt.Printf("  %s\n", p.OverallVoiceDNA)
	fmt.Println()
	if len(p.KeyPhrases) > 0 {
		fmt.Println("Key Phrases:")
		limit := 10
		if len(p.KeyPhrases) < limit {
			limit = len(p.KeyPhrases)
		}
		for _, phrase := range p.KeyPhrases[:limit] {
			fmt.Printf("  • \"%s\"\n", phrase)
		}
	}
}

// PrintFull displays the complete profile.
func PrintFull(p *analyzer.StyleProfile) {
	fmt.Println("Voice Style Profile")
	fmt.Println(strings.Repeat("═", 50))
	fmt.Printf("Analyzed:  %s\n", formatTime(p.AnalyzedAt))
	fmt.Printf("Samples:   %d\n", p.SampleCount)
	fmt.Printf("Words:     %d\n", p.TotalWords)
	fmt.Println()

	fmt.Println("Voice DNA")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("  %s\n\n", p.OverallVoiceDNA)

	fmt.Println("Vocabulary")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("  Register:            %s\n", p.Vocabulary.PreferredRegister)
	fmt.Printf("  Avg sentence length: %.0f words\n", p.Vocabulary.AvgSentenceLength)
	if len(p.Vocabulary.CommonSlang) > 0 {
		fmt.Printf("  Slang:               %s\n", strings.Join(p.Vocabulary.CommonSlang, ", "))
	}
	if len(p.Vocabulary.FillerWords) > 0 {
		fmt.Printf("  Filler words:        %s\n", strings.Join(p.Vocabulary.FillerWords, ", "))
	}
	if len(p.Vocabulary.Contractions) > 0 {
		fmt.Printf("  Contractions:        %s\n", strings.Join(p.Vocabulary.Contractions, ", "))
	}
	fmt.Println()

	fmt.Println("Humor")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("  Style:     %s\n", p.Humor.Style)
	fmt.Printf("  Frequency: %s\n", p.Humor.Frequency)
	fmt.Printf("  %s\n", p.Humor.Description)
	fmt.Println()

	fmt.Println("Argument Style")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("  Pattern:    %s\n", p.ArgumentStyle.Pattern)
	fmt.Printf("  Persuasion: %s\n", p.ArgumentStyle.Persuasion)
	fmt.Printf("  %s\n", p.ArgumentStyle.Description)
	fmt.Println()

	fmt.Println("Emotional Range")
	fmt.Println(strings.Repeat("─", 50))
	if len(p.EmotionalRange.PrimaryTones) > 0 {
		fmt.Printf("  Tones:     %s\n", strings.Join(p.EmotionalRange.PrimaryTones, ", "))
	}
	fmt.Printf("  Intensity: %s\n", p.EmotionalRange.Intensity)
	fmt.Printf("  %s\n", p.EmotionalRange.Description)
	fmt.Println()

	fmt.Println("Rhythm & Cadence")
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("  Pacing:    %s\n", p.Rhythm.Pacing)
	fmt.Printf("  Variation: %s\n", p.Rhythm.SentenceVariation)
	if len(p.Rhythm.Patterns) > 0 {
		fmt.Println("  Patterns:")
		for _, pat := range p.Rhythm.Patterns {
			fmt.Printf("    • %s\n", pat)
		}
	}
	fmt.Println()

	fmt.Println("Key Phrases")
	fmt.Println(strings.Repeat("─", 50))
	for _, phrase := range p.KeyPhrases {
		fmt.Printf("  • \"%s\"\n", phrase)
	}
	fmt.Println()

	fmt.Println("Avoid List")
	fmt.Println(strings.Repeat("─", 50))
	for _, word := range p.AvoidList {
		fmt.Printf("  ✗ %s\n", word)
	}
}

func formatTime(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.Format("2006-01-02 15:04")
}
