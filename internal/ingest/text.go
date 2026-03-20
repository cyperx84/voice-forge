package ingest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cyperx84/voice-forge/internal/corpus"
	"github.com/google/uuid"
)

// TextOptions configures text ingestion.
type TextOptions struct {
	Source string
	Format string // plain, discord-export, twitter-archive
	Tags   []string
}

// IngestTextFile reads a text/markdown file and adds it to the corpus.
func IngestTextFile(db *corpus.DB, corpusRoot, filePath string, opts TextOptions) (*corpus.Item, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return nil, fmt.Errorf("file is empty: %s", filePath)
	}

	source := opts.Source
	if source == "" {
		source = "local"
	}

	// Copy file to corpus
	id := uuid.New().String()
	ext := filepath.Ext(filePath)
	if ext == "" {
		ext = ".md"
	}
	destDir := filepath.Join(corpusRoot, "text")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, err
	}
	destPath := filepath.Join(destDir, id+ext)
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return nil, err
	}

	info, _ := os.Stat(filePath)
	createdAt := time.Now().Format(time.RFC3339)
	if info != nil {
		createdAt = info.ModTime().Format(time.RFC3339)
	}

	item := &corpus.Item{
		ID:         id,
		Type:       corpus.TypeText,
		Source:     source,
		CreatedAt:  createdAt,
		IngestedAt: time.Now().Format(time.RFC3339),
		Path:       filepath.Join("text", id+ext),
		Transcript: text,
		Tags:       opts.Tags,
		WordCount:  len(strings.Fields(text)),
		FileSize:   int64(len(data)),
	}

	if err := db.Insert(item); err != nil {
		return nil, fmt.Errorf("inserting into db: %w", err)
	}

	return item, nil
}

// IngestTextString ingests raw text content (e.g. from stdin).
func IngestTextString(db *corpus.DB, corpusRoot, text, source string, tags []string) (*corpus.Item, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("empty text")
	}

	if source == "" {
		source = "note"
	}

	id := uuid.New().String()
	destDir := filepath.Join(corpusRoot, "text")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, err
	}
	destPath := filepath.Join(destDir, id+".md")
	if err := os.WriteFile(destPath, []byte(text), 0644); err != nil {
		return nil, err
	}

	item := &corpus.Item{
		ID:         id,
		Type:       corpus.TypeText,
		Source:     source,
		CreatedAt:  time.Now().Format(time.RFC3339),
		IngestedAt: time.Now().Format(time.RFC3339),
		Path:       filepath.Join("text", id+".md"),
		Transcript: text,
		Tags:       tags,
		WordCount:  len(strings.Fields(text)),
		FileSize:   int64(len(text)),
	}

	if err := db.Insert(item); err != nil {
		return nil, fmt.Errorf("inserting into db: %w", err)
	}

	return item, nil
}
