package ingest

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Tweet represents a tweet from a Twitter archive.
type Tweet struct {
	Tweet struct {
		ID        string `json:"id_str"`
		FullText  string `json:"full_text"`
		CreatedAt string `json:"created_at"`
	} `json:"tweet"`
}

// ParseTwitterArchive reads a Twitter archive JSON and returns SocialPost items.
func ParseTwitterArchive(filePath string) ([]SocialPost, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading twitter archive: %w", err)
	}

	// Try as simple flat array first (most common export format)
	var simpleTweets []struct {
		ID        string `json:"id_str"`
		FullText  string `json:"full_text"`
		Text      string `json:"text"`
		CreatedAt string `json:"created_at"`
	}
	var tweets []Tweet

	if err := json.Unmarshal(data, &simpleTweets); err == nil {
		// Check if we got any content (flat format may parse but be empty for wrapped format)
		hasContent := false
		for _, t := range simpleTweets {
			if t.FullText != "" || t.Text != "" {
				hasContent = true
				break
			}
		}
		if hasContent {
			for _, t := range simpleTweets {
				text := t.FullText
				if text == "" {
					text = t.Text
				}
				tw := Tweet{}
				tw.Tweet.ID = t.ID
				tw.Tweet.FullText = text
				tw.Tweet.CreatedAt = t.CreatedAt
				tweets = append(tweets, tw)
			}
		} else {
			// Try wrapped format: [{"tweet": {...}}]
			if err := json.Unmarshal(data, &tweets); err != nil {
				return nil, fmt.Errorf("parsing twitter archive: %w", err)
			}
		}
	} else {
		// Try wrapped format
		if err := json.Unmarshal(data, &tweets); err != nil {
			return nil, fmt.Errorf("parsing twitter archive: %w", err)
		}
	}

	var posts []SocialPost
	for _, t := range tweets {
		text := t.Tweet.FullText
		if text == "" {
			continue
		}

		ts := t.Tweet.CreatedAt
		if ts == "" {
			ts = time.Now().Format(time.RFC3339)
		}

		posts = append(posts, SocialPost{
			Text:      text,
			Timestamp: ts,
			Source:    "twitter",
		})
	}

	return posts, nil
}
