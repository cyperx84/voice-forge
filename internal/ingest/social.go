package ingest

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cyperx84/voice-forge/internal/corpus"
	"github.com/google/uuid"
)

// SocialPost represents a generic social media post.
type SocialPost struct {
	Text      string `json:"text"`
	Author    string `json:"author"`
	Timestamp string `json:"timestamp"`
	Source    string `json:"source"`
	URL      string `json:"url"`
}

// IngestSocialPosts ingests a slice of social posts into the corpus.
func IngestSocialPosts(db *corpus.DB, corpusRoot string, posts []SocialPost, source string) ([]*corpus.Item, error) {
	destDir := filepath.Join(corpusRoot, "social")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, err
	}

	var items []*corpus.Item
	for _, post := range posts {
		text := strings.TrimSpace(post.Text)
		if text == "" {
			continue
		}

		id := uuid.New().String()
		postData, _ := json.Marshal(post)
		destPath := filepath.Join(destDir, id+".json")
		if err := os.WriteFile(destPath, postData, 0644); err != nil {
			continue
		}

		s := source
		if s == "" {
			s = post.Source
		}
		if s == "" {
			s = "social"
		}

		createdAt := post.Timestamp
		if createdAt == "" {
			createdAt = time.Now().Format(time.RFC3339)
		}

		metadata := map[string]string{}
		if post.Author != "" {
			metadata["author"] = post.Author
		}
		if post.URL != "" {
			metadata["url"] = post.URL
		}

		item := &corpus.Item{
			ID:         id,
			Type:       corpus.TypeSocial,
			Source:     s,
			CreatedAt:  createdAt,
			IngestedAt: time.Now().Format(time.RFC3339),
			Path:       filepath.Join("social", id+".json"),
			Transcript: text,
			Metadata:   metadata,
			WordCount:  len(strings.Fields(text)),
			FileSize:   int64(len(postData)),
		}

		if err := db.Insert(item); err != nil {
			continue
		}
		items = append(items, item)
	}

	return items, nil
}

// IngestSocialJSON ingests a JSON file containing an array of SocialPost objects.
func IngestSocialJSON(db *corpus.DB, corpusRoot, filePath, source string) ([]*corpus.Item, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var posts []SocialPost
	if err := json.Unmarshal(data, &posts); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	return IngestSocialPosts(db, corpusRoot, posts, source)
}
