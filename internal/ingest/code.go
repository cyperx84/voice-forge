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

// CodeOptions configures code ingestion.
type CodeOptions struct {
	Source   string
	Language string
	Tags     []string
}

var codeExts = map[string]string{
	".go":   "go",
	".py":   "python",
	".ts":   "typescript",
	".tsx":  "typescript",
	".js":   "javascript",
	".jsx":  "javascript",
	".rs":   "rust",
	".java": "java",
	".c":    "c",
	".cpp":  "cpp",
	".h":    "c",
	".rb":   "ruby",
	".sh":   "shell",
	".lua":  "lua",
	".zig":  "zig",
	".swift": "swift",
	".kt":   "kotlin",
}

// DetectLanguage returns the language for a file extension.
func DetectLanguage(ext string) string {
	if lang, ok := codeExts[strings.ToLower(ext)]; ok {
		return lang
	}
	return ""
}

// IsCodeFile checks if a file is a supported code file.
func IsCodeFile(name string) bool {
	return DetectLanguage(filepath.Ext(name)) != ""
}

// IngestCodeFile ingests a single code file.
func IngestCodeFile(db *corpus.DB, corpusRoot, filePath string, opts CodeOptions) (*corpus.Item, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	content := string(data)

	lang := opts.Language
	if lang == "" {
		lang = DetectLanguage(filepath.Ext(filePath))
	}

	source := opts.Source
	if source == "" {
		source = "local"
	}

	id := uuid.New().String()
	ext := filepath.Ext(filePath)
	destDir := filepath.Join(corpusRoot, "code")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, err
	}
	destPath := filepath.Join(destDir, id+ext)
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return nil, err
	}

	metadata := map[string]string{
		"language":      lang,
		"original_name": filepath.Base(filePath),
	}

	item := &corpus.Item{
		ID:         id,
		Type:       corpus.TypeCode,
		Source:     source,
		CreatedAt:  info.ModTime().Format(time.RFC3339),
		IngestedAt: time.Now().Format(time.RFC3339),
		Path:       filepath.Join("code", id+ext),
		Transcript: content,
		Tags:       opts.Tags,
		Metadata:   metadata,
		WordCount:  len(strings.Fields(content)),
		FileSize:   info.Size(),
	}

	if err := db.Insert(item); err != nil {
		return nil, fmt.Errorf("inserting into db: %w", err)
	}

	return item, nil
}

// IngestCodeDir scans a directory for code files and ingests them.
func IngestCodeDir(db *corpus.DB, corpusRoot, dirPath string, opts CodeOptions) ([]*corpus.Item, error) {
	var items []*corpus.Item

	// Skip common non-source directories
	skipDirs := map[string]bool{
		".git": true, "node_modules": true, "vendor": true, "__pycache__": true,
		".venv": true, "target": true, "build": true, "dist": true, ".next": true,
	}

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if skipDirs[info.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		if !IsCodeFile(info.Name()) {
			return nil
		}

		// Filter by language if specified
		if opts.Language != "" {
			lang := DetectLanguage(filepath.Ext(info.Name()))
			if lang != opts.Language {
				return nil
			}
		}

		// Skip very large files (>100KB)
		if info.Size() > 100*1024 {
			return nil
		}

		item, err := IngestCodeFile(db, corpusRoot, path, opts)
		if err != nil {
			return nil
		}
		items = append(items, item)
		return nil
	})

	return items, err
}
