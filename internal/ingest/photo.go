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

// PhotoOptions configures photo ingestion.
type PhotoOptions struct {
	Source    string
	Tags     []string
	Recursive bool
}

var photoExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".webp": true,
	".gif": true, ".bmp": true, ".tiff": true, ".heic": true,
}

// IsPhotoFile checks if a file is a supported photo format.
func IsPhotoFile(name string) bool {
	return photoExts[strings.ToLower(filepath.Ext(name))]
}

// IngestPhotoFile copies a photo into the corpus and indexes it.
func IngestPhotoFile(db *corpus.DB, corpusRoot, filePath string, opts PhotoOptions) (*corpus.Item, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	source := opts.Source
	if source == "" {
		source = "local"
	}

	id := uuid.New().String()
	ext := filepath.Ext(filePath)
	destDir := filepath.Join(corpusRoot, "photos")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, err
	}

	// Copy photo
	destPath := filepath.Join(destDir, id+ext)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading photo: %w", err)
	}
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return nil, err
	}

	metadata := map[string]string{
		"original_name": filepath.Base(filePath),
	}

	item := &corpus.Item{
		ID:         id,
		Type:       corpus.TypePhoto,
		Source:     source,
		CreatedAt:  info.ModTime().Format(time.RFC3339),
		IngestedAt: time.Now().Format(time.RFC3339),
		Path:       filepath.Join("photos", id+ext),
		Tags:       opts.Tags,
		Metadata:   metadata,
		FileSize:   info.Size(),
	}

	if err := db.Insert(item); err != nil {
		return nil, fmt.Errorf("inserting into db: %w", err)
	}

	return item, nil
}

// IngestPhotoDir recursively ingests photos from a directory.
func IngestPhotoDir(db *corpus.DB, corpusRoot, dirPath string, opts PhotoOptions) ([]*corpus.Item, error) {
	var items []*corpus.Item
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip errors
		}
		if info.IsDir() {
			if !opts.Recursive && path != dirPath {
				return filepath.SkipDir
			}
			return nil
		}
		if !IsPhotoFile(info.Name()) {
			return nil
		}
		item, err := IngestPhotoFile(db, corpusRoot, path, opts)
		if err != nil {
			return nil // skip individual file errors
		}
		items = append(items, item)
		return nil
	})
	return items, err
}
