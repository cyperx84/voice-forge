package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// MultiSourceProfile extends StyleProfile with multi-source analysis results.
type MultiSourceProfile struct {
	StyleProfile
	WritingStyle  *WritingStyle  `json:"writing_style,omitempty"`
	CodingStyle   *CodingStyle   `json:"coding_style,omitempty"`
	ContentThemes []string       `json:"content_themes,omitempty"`
}

// WritingStyle captures patterns from text corpus.
type WritingStyle struct {
	SentenceStructure string   `json:"sentence_structure"`
	Vocabulary        string   `json:"vocabulary"`
	Formatting        string   `json:"formatting_preferences"`
	Tone              string   `json:"tone"`
	Patterns          []string `json:"patterns"`
}

// CodingStyle captures patterns from code corpus.
type CodingStyle struct {
	NamingConvention   string   `json:"naming_convention"`
	CommentStyle       string   `json:"comment_style"`
	Architecture       string   `json:"architecture_preferences"`
	ErrorHandling      string   `json:"error_handling"`
	Patterns           []string `json:"patterns"`
	PreferredLanguages []string `json:"preferred_languages"`
}

const writingAnalysisPrompt = `You are a writing style analyst. Analyze the following text samples from a single person and extract their writing style profile.

Output ONLY valid JSON (no markdown, no code fences):

{
  "sentence_structure": "how they structure sentences (simple/complex/varied)",
  "vocabulary": "vocabulary preferences (technical/casual/mixed)",
  "formatting_preferences": "how they format text (headers, lists, paragraphs, etc)",
  "tone": "overall writing tone",
  "patterns": ["recurring writing patterns"]
}

Text samples:

%s`

const codingAnalysisPrompt = `You are a code style analyst. Analyze the following code samples from a single developer and extract their coding style profile.

Output ONLY valid JSON (no markdown, no code fences):

{
  "naming_convention": "how they name things (camelCase/snake_case/etc)",
  "comment_style": "how/when they comment code",
  "architecture_preferences": "how they organize code",
  "error_handling": "how they handle errors",
  "patterns": ["recurring coding patterns"],
  "preferred_languages": ["languages evident in samples"]
}

Code samples:

%s`

const themesPrompt = `You are a content analyst. Analyze these text samples from across multiple sources (voice transcripts, written text, code, social posts) and identify the recurring themes this person engages with.

Output ONLY a JSON array of theme strings (no markdown, no code fences):

["theme1", "theme2", ...]

Samples:

%s`

// AnalyzeMultiSource runs analysis across all corpus types.
func AnalyzeMultiSource(
	voiceTranscripts []string,
	textSamples []string,
	codeSamples []string,
	socialSamples []string,
	llmCommand string,
	llmArgs []string,
) (*MultiSourceProfile, error) {
	// Start with voice analysis (existing)
	var baseProfile *StyleProfile
	if len(voiceTranscripts) > 0 {
		var err error
		baseProfile, err = Analyze(voiceTranscripts, llmCommand, llmArgs)
		if err != nil {
			return nil, fmt.Errorf("voice analysis: %w", err)
		}
	} else {
		baseProfile = &StyleProfile{
			AnalyzedAt: time.Now().Format(time.RFC3339),
		}
	}

	multi := &MultiSourceProfile{
		StyleProfile: *baseProfile,
	}

	// Writing style analysis
	if len(textSamples) > 0 {
		fmt.Println("Analyzing writing style...")
		ws, err := analyzeWritingStyle(textSamples, llmCommand, llmArgs)
		if err != nil {
			fmt.Printf("  warning: writing analysis failed: %v\n", err)
		} else {
			multi.WritingStyle = ws
		}
	}

	// Coding style analysis
	if len(codeSamples) > 0 {
		fmt.Println("Analyzing coding style...")
		cs, err := analyzeCodingStyle(codeSamples, llmCommand, llmArgs)
		if err != nil {
			fmt.Printf("  warning: coding analysis failed: %v\n", err)
		} else {
			multi.CodingStyle = cs
		}
	}

	// Content themes from all sources
	var allSamples []string
	allSamples = append(allSamples, voiceTranscripts...)
	allSamples = append(allSamples, textSamples...)
	allSamples = append(allSamples, socialSamples...)
	if len(allSamples) > 0 {
		fmt.Println("Extracting content themes...")
		themes, err := extractThemes(allSamples, llmCommand, llmArgs)
		if err != nil {
			fmt.Printf("  warning: theme extraction failed: %v\n", err)
		} else {
			multi.ContentThemes = themes
		}
	}

	return multi, nil
}

func analyzeWritingStyle(samples []string, llmCommand string, llmArgs []string) (*WritingStyle, error) {
	corpus := joinAndTruncate(samples, maxCorpusBytes)
	prompt := fmt.Sprintf(writingAnalysisPrompt, corpus)

	raw, err := runLLM(prompt, llmCommand, llmArgs)
	if err != nil {
		return nil, err
	}

	var ws WritingStyle
	if err := json.Unmarshal([]byte(raw), &ws); err != nil {
		return nil, fmt.Errorf("parsing writing style: %w", err)
	}
	return &ws, nil
}

func analyzeCodingStyle(samples []string, llmCommand string, llmArgs []string) (*CodingStyle, error) {
	corpus := joinAndTruncate(samples, maxCorpusBytes)
	prompt := fmt.Sprintf(codingAnalysisPrompt, corpus)

	raw, err := runLLM(prompt, llmCommand, llmArgs)
	if err != nil {
		return nil, err
	}

	var cs CodingStyle
	if err := json.Unmarshal([]byte(raw), &cs); err != nil {
		return nil, fmt.Errorf("parsing coding style: %w", err)
	}
	return &cs, nil
}

func extractThemes(samples []string, llmCommand string, llmArgs []string) ([]string, error) {
	corpus := joinAndTruncate(samples, maxCorpusBytes)
	prompt := fmt.Sprintf(themesPrompt, corpus)

	raw, err := runLLM(prompt, llmCommand, llmArgs)
	if err != nil {
		return nil, err
	}

	var themes []string
	if err := json.Unmarshal([]byte(raw), &themes); err != nil {
		return nil, fmt.Errorf("parsing themes: %w", err)
	}
	return themes, nil
}

func runLLM(prompt, llmCommand string, llmArgs []string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, llmCommand, llmArgs...)
	cmd.Stderr = os.Stderr
	cmd.Stdin = strings.NewReader(prompt)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("LLM command failed: %w", err)
	}

	raw := strings.TrimSpace(string(output))
	raw = stripCodeFences(raw)

	// Try to extract JSON from mixed output
	start := strings.Index(raw, "{")
	arrStart := strings.Index(raw, "[")
	if arrStart >= 0 && (start < 0 || arrStart < start) {
		end := strings.LastIndex(raw, "]")
		if end > arrStart {
			return raw[arrStart : end+1], nil
		}
	}
	if start >= 0 {
		end := strings.LastIndex(raw, "}")
		if end > start {
			return raw[start : end+1], nil
		}
	}

	return raw, nil
}

func joinAndTruncate(samples []string, maxBytes int) string {
	joined := strings.Join(samples, "\n\n---\n\n")
	if len(joined) > maxBytes {
		joined = joined[:maxBytes]
	}
	return joined
}

// SaveMultiSourceProfile writes the multi-source profile to disk.
func SaveMultiSourceProfile(profile *MultiSourceProfile, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}

	jsonPath := fmt.Sprintf("%s/style.json", outputDir)
	data, err := json.MarshalIndent(profile, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		return err
	}
	fmt.Printf("Wrote multi-source profile to %s\n", jsonPath)

	// Also write the summary markdown
	summary := generateSummary(&profile.StyleProfile)
	if profile.WritingStyle != nil {
		summary += "\n## Writing Style\n\n"
		summary += fmt.Sprintf("- **Structure:** %s\n", profile.WritingStyle.SentenceStructure)
		summary += fmt.Sprintf("- **Vocabulary:** %s\n", profile.WritingStyle.Vocabulary)
		summary += fmt.Sprintf("- **Formatting:** %s\n", profile.WritingStyle.Formatting)
		summary += fmt.Sprintf("- **Tone:** %s\n", profile.WritingStyle.Tone)
	}
	if profile.CodingStyle != nil {
		summary += "\n## Coding Style\n\n"
		summary += fmt.Sprintf("- **Naming:** %s\n", profile.CodingStyle.NamingConvention)
		summary += fmt.Sprintf("- **Comments:** %s\n", profile.CodingStyle.CommentStyle)
		summary += fmt.Sprintf("- **Architecture:** %s\n", profile.CodingStyle.Architecture)
		summary += fmt.Sprintf("- **Error handling:** %s\n", profile.CodingStyle.ErrorHandling)
	}
	if len(profile.ContentThemes) > 0 {
		summary += "\n## Content Themes\n\n"
		for _, theme := range profile.ContentThemes {
			summary += fmt.Sprintf("- %s\n", theme)
		}
	}

	mdPath := fmt.Sprintf("%s/style-summary.md", outputDir)
	if err := os.WriteFile(mdPath, []byte(summary), 0644); err != nil {
		return err
	}
	fmt.Printf("Wrote style summary to %s\n", mdPath)

	return nil
}
