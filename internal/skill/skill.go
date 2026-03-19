package skill

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cyperx84/voice-forge/internal/analyzer"
)

// Generate creates OpenClaw skill files from a style profile.
func Generate(profile *analyzer.StyleProfile, outputDir string) error {
	refsDir := filepath.Join(outputDir, "references")
	if err := os.MkdirAll(refsDir, 0755); err != nil {
		return fmt.Errorf("creating skill directories: %w", err)
	}

	// Generate SKILL.md
	skillMd := generateSkillMd(profile)
	if err := os.WriteFile(filepath.Join(outputDir, "SKILL.md"), []byte(skillMd), 0644); err != nil {
		return fmt.Errorf("writing SKILL.md: %w", err)
	}

	// Generate references/voice-profile.md
	voiceProfile := generateVoiceProfile(profile)
	if err := os.WriteFile(filepath.Join(refsDir, "voice-profile.md"), []byte(voiceProfile), 0644); err != nil {
		return fmt.Errorf("writing voice-profile.md: %w", err)
	}

	// Generate references/avoid-list.md
	avoidList := generateAvoidList(profile)
	if err := os.WriteFile(filepath.Join(refsDir, "avoid-list.md"), []byte(avoidList), 0644); err != nil {
		return fmt.Errorf("writing avoid-list.md: %w", err)
	}

	// Generate references/key-phrases.md
	keyPhrases := generateKeyPhrases(profile)
	if err := os.WriteFile(filepath.Join(refsDir, "key-phrases.md"), []byte(keyPhrases), 0644); err != nil {
		return fmt.Errorf("writing key-phrases.md: %w", err)
	}

	return nil
}

func generateSkillMd(p *analyzer.StyleProfile) string {
	var b strings.Builder

	b.WriteString(`---
name: cyperx-voice
description: Write and communicate in CyperX's authentic voice and style
---

# CyperX Voice Skill

You are writing as CyperX. Your goal is to produce text that is indistinguishable from CyperX's natural voice.

## Instructions

1. **Read** ` + "`references/voice-profile.md`" + ` to understand the full voice profile — vocabulary, humor, rhythm, emotional range, and argument style.
2. **Use** key phrases from ` + "`references/key-phrases.md`" + ` naturally in your writing. Don't force them — weave them in where they fit.
3. **Never** use words or phrases from ` + "`references/avoid-list.md`" + `. These would immediately sound fake.
4. **Match** the rhythm and cadence described in the profile. Pay attention to sentence length variation and pacing.

## Voice DNA

`)
	b.WriteString(p.OverallVoiceDNA)
	b.WriteString("\n\n## Quick Reference\n\n")

	b.WriteString(fmt.Sprintf("- **Register:** %s\n", p.Vocabulary.PreferredRegister))
	b.WriteString(fmt.Sprintf("- **Avg sentence length:** %.0f words\n", p.Vocabulary.AvgSentenceLength))
	b.WriteString(fmt.Sprintf("- **Humor:** %s (%s)\n", p.Humor.Style, p.Humor.Frequency))
	b.WriteString(fmt.Sprintf("- **Pacing:** %s\n", p.Rhythm.Pacing))
	b.WriteString(fmt.Sprintf("- **Emotional intensity:** %s\n", p.EmotionalRange.Intensity))
	b.WriteString(fmt.Sprintf("- **Argument style:** %s\n", p.ArgumentStyle.Pattern))

	if len(p.Vocabulary.FillerWords) > 0 {
		b.WriteString(fmt.Sprintf("- **Filler words:** %s\n", strings.Join(p.Vocabulary.FillerWords, ", ")))
	}
	if len(p.Vocabulary.CommonSlang) > 0 {
		b.WriteString(fmt.Sprintf("- **Common slang:** %s\n", strings.Join(p.Vocabulary.CommonSlang, ", ")))
	}

	b.WriteString("\n## Key Rules\n\n")
	b.WriteString("- Write as if speaking out loud — this voice comes from voice messages, not essays\n")
	b.WriteString("- Use contractions naturally\n")
	b.WriteString("- Don't over-polish — some roughness is authentic\n")
	b.WriteString("- Match the emotional tone to the topic\n")
	b.WriteString("- When in doubt, be direct rather than formal\n")

	return b.String()
}

func generateVoiceProfile(p *analyzer.StyleProfile) string {
	var b strings.Builder

	b.WriteString("# Voice Profile\n\n")
	b.WriteString(fmt.Sprintf("*Based on %d samples, %d words*\n\n", p.SampleCount, p.TotalWords))

	b.WriteString("## Voice DNA\n\n")
	b.WriteString(p.OverallVoiceDNA + "\n\n")

	b.WriteString("## Vocabulary\n\n")
	b.WriteString(fmt.Sprintf("- **Preferred register:** %s\n", p.Vocabulary.PreferredRegister))
	b.WriteString(fmt.Sprintf("- **Average sentence length:** %.0f words\n", p.Vocabulary.AvgSentenceLength))
	if len(p.Vocabulary.CommonSlang) > 0 {
		b.WriteString(fmt.Sprintf("- **Common slang:** %s\n", strings.Join(p.Vocabulary.CommonSlang, ", ")))
	}
	if len(p.Vocabulary.Contractions) > 0 {
		b.WriteString(fmt.Sprintf("- **Contractions:** %s\n", strings.Join(p.Vocabulary.Contractions, ", ")))
	}
	if len(p.Vocabulary.FillerWords) > 0 {
		b.WriteString(fmt.Sprintf("- **Filler words:** %s\n", strings.Join(p.Vocabulary.FillerWords, ", ")))
	}
	if len(p.Vocabulary.TechnicalTerms) > 0 {
		b.WriteString(fmt.Sprintf("- **Technical terms:** %s\n", strings.Join(p.Vocabulary.TechnicalTerms, ", ")))
	}
	b.WriteString("\n")

	b.WriteString("## Humor\n\n")
	b.WriteString(fmt.Sprintf("- **Style:** %s\n", p.Humor.Style))
	b.WriteString(fmt.Sprintf("- **Frequency:** %s\n", p.Humor.Frequency))
	b.WriteString(fmt.Sprintf("- **Description:** %s\n", p.Humor.Description))
	if len(p.Humor.Examples) > 0 {
		b.WriteString("- **Examples:**\n")
		for _, ex := range p.Humor.Examples {
			b.WriteString(fmt.Sprintf("  - \"%s\"\n", ex))
		}
	}
	b.WriteString("\n")

	b.WriteString("## Argument Style\n\n")
	b.WriteString(fmt.Sprintf("- **Pattern:** %s\n", p.ArgumentStyle.Pattern))
	b.WriteString(fmt.Sprintf("- **Persuasion:** %s\n", p.ArgumentStyle.Persuasion))
	b.WriteString(fmt.Sprintf("- **Description:** %s\n", p.ArgumentStyle.Description))
	if len(p.ArgumentStyle.Transitions) > 0 {
		b.WriteString(fmt.Sprintf("- **Transitions:** %s\n", strings.Join(p.ArgumentStyle.Transitions, ", ")))
	}
	b.WriteString("\n")

	b.WriteString("## Emotional Range\n\n")
	if len(p.EmotionalRange.PrimaryTones) > 0 {
		b.WriteString(fmt.Sprintf("- **Primary tones:** %s\n", strings.Join(p.EmotionalRange.PrimaryTones, ", ")))
	}
	b.WriteString(fmt.Sprintf("- **Intensity:** %s\n", p.EmotionalRange.Intensity))
	b.WriteString(fmt.Sprintf("- **Description:** %s\n", p.EmotionalRange.Description))
	if len(p.EmotionalRange.Triggers) > 0 {
		b.WriteString(fmt.Sprintf("- **Triggers:** %s\n", strings.Join(p.EmotionalRange.Triggers, ", ")))
	}
	b.WriteString("\n")

	b.WriteString("## Rhythm & Cadence\n\n")
	b.WriteString(fmt.Sprintf("- **Pacing:** %s\n", p.Rhythm.Pacing))
	b.WriteString(fmt.Sprintf("- **Sentence variation:** %s\n", p.Rhythm.SentenceVariation))
	b.WriteString(fmt.Sprintf("- **Description:** %s\n", p.Rhythm.Description))
	if len(p.Rhythm.Patterns) > 0 {
		b.WriteString("- **Patterns:**\n")
		for _, pat := range p.Rhythm.Patterns {
			b.WriteString(fmt.Sprintf("  - %s\n", pat))
		}
	}

	return b.String()
}

func generateAvoidList(p *analyzer.StyleProfile) string {
	var b strings.Builder

	b.WriteString("# Avoid List\n\n")
	b.WriteString("These words and phrases would sound unnatural for CyperX. Never use them.\n\n")

	if len(p.AvoidList) == 0 {
		b.WriteString("*No avoid list entries yet.*\n")
		return b.String()
	}

	for _, word := range p.AvoidList {
		b.WriteString(fmt.Sprintf("- %s\n", word))
	}

	return b.String()
}

func generateKeyPhrases(p *analyzer.StyleProfile) string {
	var b strings.Builder

	b.WriteString("# Key Phrases\n\n")
	b.WriteString("These are CyperX's signature phrases and expressions. Use them naturally.\n\n")

	if len(p.KeyPhrases) == 0 {
		b.WriteString("*No key phrases extracted yet.*\n")
		return b.String()
	}

	for _, phrase := range p.KeyPhrases {
		b.WriteString(fmt.Sprintf("- \"%s\"\n", phrase))
	}

	return b.String()
}
