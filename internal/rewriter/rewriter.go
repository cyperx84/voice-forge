package rewriter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/cyperx84/voice-forge/internal/character"
)

const rewritePrompt = `You are a voice style rewriter. Your job is to take input text and rewrite it in a specific character's voice.

Here is the base voice style profile (JSON):
%s

Here is the character to write as:
- Name: %s
- Description: %s
- Register: %s
- Pacing: %s
- Persona: %s
- Emoji style: %s
- Vocabulary to use: %s
- Words to avoid: %s

Rewrite the following text in this character's voice. Keep the core meaning but transform the tone, word choice, and pacing to match the character. Output ONLY the rewritten text, nothing else.

Text to rewrite:
%s`

const generatePrompt = `You are a voice style writer. Your job is to generate text on a given topic in a specific character's voice.

Here is the base voice style profile (JSON):
%s

Here is the character to write as:
- Name: %s
- Description: %s
- Register: %s
- Pacing: %s
- Persona: %s
- Emoji style: %s
- Vocabulary to use: %s
- Words to avoid: %s

Write a short piece (2-4 paragraphs) about the following topic in this character's voice. Match the tone, word choice, pacing, and personality. Output ONLY the generated text, nothing else.

Topic: %s`

// Rewrite transforms text through a character's voice using the LLM.
func Rewrite(text string, ch *character.Character, styleJSON string, llmCommand string, llmArgs []string) (string, error) {
	prompt := fmt.Sprintf(rewritePrompt,
		styleJSON,
		ch.Name,
		ch.Description,
		ch.ToneShift.Register,
		ch.ToneShift.Pacing,
		ch.ToneShift.Persona,
		ch.ToneShift.EmojiStyle,
		strings.Join(ch.ToneShift.Vocabulary, ", "),
		strings.Join(ch.ToneShift.AvoidWords, ", "),
		text,
	)

	return runLLM(prompt, llmCommand, llmArgs)
}

// Generate creates new text on a topic in a character's voice using the LLM.
func Generate(topic string, ch *character.Character, styleJSON string, llmCommand string, llmArgs []string) (string, error) {
	prompt := fmt.Sprintf(generatePrompt,
		styleJSON,
		ch.Name,
		ch.Description,
		ch.ToneShift.Register,
		ch.ToneShift.Pacing,
		ch.ToneShift.Persona,
		ch.ToneShift.EmojiStyle,
		strings.Join(ch.ToneShift.Vocabulary, ", "),
		strings.Join(ch.ToneShift.AvoidWords, ", "),
		topic,
	)

	return runLLM(prompt, llmCommand, llmArgs)
}

// LoadStyleJSON reads the style profile JSON from disk.
func LoadStyleJSON(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading style profile: %w", err)
	}

	// Validate it's valid JSON
	var raw json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return "", fmt.Errorf("invalid style profile JSON: %w", err)
	}

	return string(data), nil
}

func runLLM(prompt string, llmCommand string, llmArgs []string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, llmCommand, llmArgs...)
	cmd.Stderr = os.Stderr
	cmd.Stdin = strings.NewReader(prompt)

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("LLM command failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
