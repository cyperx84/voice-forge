package ingest

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// DiscordMessage represents a message from a Discord data export.
type DiscordMessage struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
	Author    struct {
		Name string `json:"name"`
		ID   string `json:"id"`
	} `json:"author"`
}

// DiscordExport represents a Discord channel export JSON file.
type DiscordExport struct {
	Messages []DiscordMessage `json:"messages"`
	Channel  struct {
		Name string `json:"name"`
	} `json:"channel"`
}

// ParseDiscordExport reads a Discord export JSON and returns SocialPost items.
func ParseDiscordExport(filePath string) ([]SocialPost, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading discord export: %w", err)
	}

	// Try as a direct array of messages first
	var messages []DiscordMessage
	if err := json.Unmarshal(data, &messages); err != nil {
		// Try as a DiscordExport object
		var export DiscordExport
		if err := json.Unmarshal(data, &export); err != nil {
			return nil, fmt.Errorf("parsing discord export: %w", err)
		}
		messages = export.Messages
	}

	var posts []SocialPost
	for _, msg := range messages {
		if msg.Content == "" {
			continue
		}

		ts := msg.Timestamp
		if ts == "" {
			ts = time.Now().Format(time.RFC3339)
		}

		posts = append(posts, SocialPost{
			Text:      msg.Content,
			Author:    msg.Author.Name,
			Timestamp: ts,
			Source:    "discord",
		})
	}

	return posts, nil
}
