package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// maxCorpusBytes caps the corpus size piped to the LLM (~50K tokens ≈ 200KB).
const maxCorpusBytes = 200_000

// StyleProfile is the structured output from LLM analysis.
type StyleProfile struct {
	AnalyzedAt       string            `json:"analyzed_at"`
	SampleCount      int               `json:"sample_count"`
	TotalWords       int               `json:"total_words"`
	Vocabulary       VocabularyProfile `json:"vocabulary"`
	Humor            HumorProfile      `json:"humor"`
	ArgumentStyle    ArgumentProfile   `json:"argument_style"`
	EmotionalRange   EmotionalProfile  `json:"emotional_range"`
	Rhythm           RhythmProfile     `json:"rhythm"`
	KeyPhrases       []string          `json:"key_phrases"`
	AvoidList        []string          `json:"avoid_list"`
	OverallVoiceDNA  string            `json:"overall_voice_dna"`
}

type VocabularyProfile struct {
	CommonSlang       []string `json:"common_slang"`
	Contractions      []string `json:"contractions"`
	FillerWords       []string `json:"filler_words"`
	TechnicalTerms    []string `json:"technical_terms"`
	AvgSentenceLength float64  `json:"avg_sentence_length_words"`
	PreferredRegister string   `json:"preferred_register"`
}

type HumorProfile struct {
	Style       string   `json:"style"`
	Frequency   string   `json:"frequency"`
	Examples    []string `json:"examples"`
	Description string   `json:"description"`
}

type ArgumentProfile struct {
	Pattern     string   `json:"pattern"`
	Transitions []string `json:"transitions"`
	Persuasion  string   `json:"persuasion_style"`
	Description string   `json:"description"`
}

type EmotionalProfile struct {
	PrimaryTones  []string `json:"primary_tones"`
	Triggers      []string `json:"triggers"`
	Intensity     string   `json:"intensity"`
	Description   string   `json:"description"`
}

type RhythmProfile struct {
	SentenceVariation string   `json:"sentence_variation"`
	Pacing            string   `json:"pacing"`
	Patterns          []string `json:"patterns"`
	Description       string   `json:"description"`
}

const analysisPrompt = `You are a linguistic analyst specializing in voice and writing style extraction.

I'm going to give you a corpus of voice message transcripts from a single person. These are casual, spoken messages — not written text. Your job is to extract a comprehensive style profile.

Analyze the following dimensions and output ONLY valid JSON (no markdown, no code fences, just raw JSON):

{
  "vocabulary": {
    "common_slang": ["list of slang/casual terms they use frequently"],
    "contractions": ["contractions they prefer"],
    "filler_words": ["filler words and verbal tics"],
    "technical_terms": ["technical/domain terms they use"],
    "avg_sentence_length_words": 0,
    "preferred_register": "casual/formal/mixed — describe their default register"
  },
  "humor": {
    "style": "type of humor (dry, self-deprecating, etc)",
    "frequency": "how often humor appears",
    "examples": ["actual examples from the corpus"],
    "description": "overall humor profile"
  },
  "argument_style": {
    "pattern": "how they build arguments/make points",
    "transitions": ["transition phrases they use"],
    "persuasion_style": "how they try to convince",
    "description": "overall argumentation pattern"
  },
  "emotional_range": {
    "primary_tones": ["list of dominant emotional tones"],
    "triggers": ["what topics/situations trigger emotional shifts"],
    "intensity": "how emotionally expressive they are",
    "description": "overall emotional profile"
  },
  "rhythm": {
    "sentence_variation": "how much sentence length varies",
    "pacing": "fast/measured/variable",
    "patterns": ["recurring rhythm patterns like 'short setup → punchy line → detail'"],
    "description": "overall cadence/rhythm"
  },
  "key_phrases": ["their signature phrases and expressions — things that are uniquely theirs"],
  "avoid_list": ["words and phrases they never use or would sound unnatural for them"],
  "overall_voice_dna": "A 2-3 sentence summary of what makes this person's voice distinctive. What would someone need to know to write convincingly as this person?"
}

Be thorough. Pull actual examples from the corpus. This profile will be used to generate text and speech in this person's voice.

Here is the corpus:

%s`

// Analyze runs LLM analysis on the transcripts and returns a StyleProfile.
func Analyze(transcripts []string, llmCommand string, llmArgs []string) (*StyleProfile, error) {
	corpus := strings.Join(transcripts, "\n\n---\n\n")
	if len(corpus) > maxCorpusBytes {
		fmt.Printf("Warning: corpus truncated from %d to %d bytes for LLM analysis\n", len(corpus), maxCorpusBytes)
		corpus = corpus[:maxCorpusBytes]
	}
	prompt := fmt.Sprintf(analysisPrompt, corpus)

	// Build command — pipe prompt via stdin to avoid arg length limits
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, llmCommand, llmArgs...)
	cmd.Stderr = os.Stderr
	cmd.Stdin = strings.NewReader(prompt)

	fmt.Println("Running LLM analysis... this may take a minute.")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("LLM command failed: %w", err)
	}

	// Parse the JSON response
	raw := strings.TrimSpace(string(output))
	// Strip markdown code fences if present
	raw = stripCodeFences(raw)

	var profile StyleProfile
	if err := json.Unmarshal([]byte(raw), &profile); err != nil {
		// Try to extract JSON from mixed output
		start := strings.Index(raw, "{")
		end := strings.LastIndex(raw, "}")
		if start >= 0 && end > start {
			raw = raw[start : end+1]
			if err := json.Unmarshal([]byte(raw), &profile); err != nil {
				return nil, fmt.Errorf("failed to parse LLM output as JSON: %w\nRaw output:\n%s", err, string(output))
			}
		} else {
			return nil, fmt.Errorf("failed to parse LLM output as JSON: %w\nRaw output:\n%s", err, string(output))
		}
	}

	profile.AnalyzedAt = time.Now().Format(time.RFC3339)
	profile.SampleCount = len(transcripts)

	// Count total words
	totalWords := 0
	for _, t := range transcripts {
		totalWords += len(strings.Fields(t))
	}
	profile.TotalWords = totalWords

	return &profile, nil
}

// SaveProfile writes the style profile to JSON and markdown summary files.
func SaveProfile(profile *StyleProfile, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	// Write style.json
	jsonPath := filepath.Join(outputDir, "style.json")
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		return err
	}
	fmt.Printf("Wrote style profile to %s\n", jsonPath)

	// Write style-summary.md
	mdPath := filepath.Join(outputDir, "style-summary.md")
	summary := generateSummary(profile)
	if err := os.WriteFile(mdPath, []byte(summary), 0644); err != nil {
		return err
	}
	fmt.Printf("Wrote style summary to %s\n", mdPath)

	return nil
}

func generateSummary(p *StyleProfile) string {
	var b strings.Builder

	b.WriteString("# Voice Style Profile\n\n")
	b.WriteString(fmt.Sprintf("*Analyzed: %s | Samples: %d | Words: %d*\n\n", p.AnalyzedAt, p.SampleCount, p.TotalWords))

	b.WriteString("## Voice DNA\n\n")
	b.WriteString(p.OverallVoiceDNA + "\n\n")

	b.WriteString("## Vocabulary\n\n")
	b.WriteString(fmt.Sprintf("- **Register:** %s\n", p.Vocabulary.PreferredRegister))
	b.WriteString(fmt.Sprintf("- **Avg sentence length:** %.0f words\n", p.Vocabulary.AvgSentenceLength))
	if len(p.Vocabulary.CommonSlang) > 0 {
		b.WriteString(fmt.Sprintf("- **Slang:** %s\n", strings.Join(p.Vocabulary.CommonSlang, ", ")))
	}
	if len(p.Vocabulary.FillerWords) > 0 {
		b.WriteString(fmt.Sprintf("- **Filler words:** %s\n", strings.Join(p.Vocabulary.FillerWords, ", ")))
	}
	if len(p.Vocabulary.Contractions) > 0 {
		b.WriteString(fmt.Sprintf("- **Contractions:** %s\n", strings.Join(p.Vocabulary.Contractions, ", ")))
	}
	b.WriteString("\n")

	b.WriteString("## Humor\n\n")
	b.WriteString(fmt.Sprintf("- **Style:** %s\n", p.Humor.Style))
	b.WriteString(fmt.Sprintf("- **Frequency:** %s\n", p.Humor.Frequency))
	b.WriteString(p.Humor.Description + "\n\n")

	b.WriteString("## Argument Style\n\n")
	b.WriteString(fmt.Sprintf("- **Pattern:** %s\n", p.ArgumentStyle.Pattern))
	b.WriteString(fmt.Sprintf("- **Persuasion:** %s\n", p.ArgumentStyle.Persuasion))
	b.WriteString(p.ArgumentStyle.Description + "\n\n")

	b.WriteString("## Emotional Range\n\n")
	if len(p.EmotionalRange.PrimaryTones) > 0 {
		b.WriteString(fmt.Sprintf("- **Primary tones:** %s\n", strings.Join(p.EmotionalRange.PrimaryTones, ", ")))
	}
	b.WriteString(fmt.Sprintf("- **Intensity:** %s\n", p.EmotionalRange.Intensity))
	b.WriteString(p.EmotionalRange.Description + "\n\n")

	b.WriteString("## Rhythm & Cadence\n\n")
	b.WriteString(fmt.Sprintf("- **Pacing:** %s\n", p.Rhythm.Pacing))
	b.WriteString(fmt.Sprintf("- **Variation:** %s\n", p.Rhythm.SentenceVariation))
	if len(p.Rhythm.Patterns) > 0 {
		b.WriteString("- **Patterns:**\n")
		for _, pat := range p.Rhythm.Patterns {
			b.WriteString(fmt.Sprintf("  - %s\n", pat))
		}
	}
	b.WriteString("\n")

	b.WriteString("## Key Phrases\n\n")
	for _, phrase := range p.KeyPhrases {
		b.WriteString(fmt.Sprintf("- \"%s\"\n", phrase))
	}
	b.WriteString("\n")

	b.WriteString("## Avoid List\n\n")
	b.WriteString("Words/phrases that would sound unnatural for this voice:\n\n")
	for _, word := range p.AvoidList {
		b.WriteString(fmt.Sprintf("- %s\n", word))
	}

	return b.String()
}

func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
	}
	if strings.HasSuffix(s, "```") {
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}
