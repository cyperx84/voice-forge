package corpus

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite corpus database.
type DB struct {
	db   *sql.DB
	path string
}

// OpenDB opens (or creates) the corpus SQLite database.
func OpenDB(path string) (*DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Enable WAL mode for better concurrent access
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, err
	}

	cdb := &DB{db: db, path: path}
	if err := cdb.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("migration: %w", err)
	}

	return cdb, nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) migrate() error {
	_, err := d.db.Exec(`
		CREATE TABLE IF NOT EXISTS corpus_items (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			source TEXT NOT NULL,
			created_at TEXT,
			ingested_at TEXT NOT NULL,
			path TEXT NOT NULL,
			transcript TEXT,
			tags TEXT,
			metadata TEXT,
			word_count INTEGER DEFAULT 0,
			duration_seconds REAL,
			file_size INTEGER
		);
		CREATE INDEX IF NOT EXISTS idx_type ON corpus_items(type);
		CREATE INDEX IF NOT EXISTS idx_source ON corpus_items(source);
		CREATE INDEX IF NOT EXISTS idx_created ON corpus_items(created_at);
	`)
	return err
}

// Insert adds a new corpus item to the database.
func (d *DB) Insert(item *Item) error {
	tagsJSON, _ := json.Marshal(item.Tags)
	metaJSON, _ := json.Marshal(item.Metadata)

	_, err := d.db.Exec(`
		INSERT OR REPLACE INTO corpus_items
			(id, type, source, created_at, ingested_at, path, transcript, tags, metadata, word_count, duration_seconds, file_size)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Type, item.Source, item.CreatedAt, item.IngestedAt,
		item.Path, item.Transcript, string(tagsJSON), string(metaJSON),
		item.WordCount, item.DurationSeconds, item.FileSize,
	)
	return err
}

// GetByID retrieves a single corpus item by ID.
func (d *DB) GetByID(id string) (*Item, error) {
	row := d.db.QueryRow(`SELECT id, type, source, created_at, ingested_at, path, transcript, tags, metadata, word_count, duration_seconds, file_size FROM corpus_items WHERE id = ?`, id)
	return scanItem(row)
}

// ListByType returns all items of a given type.
func (d *DB) ListByType(itemType string) ([]*Item, error) {
	rows, err := d.db.Query(`SELECT id, type, source, created_at, ingested_at, path, transcript, tags, metadata, word_count, duration_seconds, file_size FROM corpus_items WHERE type = ? ORDER BY ingested_at DESC`, itemType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanItems(rows)
}

// Recent returns the most recently ingested items.
func (d *DB) Recent(limit int) ([]*Item, error) {
	rows, err := d.db.Query(`SELECT id, type, source, created_at, ingested_at, path, transcript, tags, metadata, word_count, duration_seconds, file_size FROM corpus_items ORDER BY ingested_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanItems(rows)
}

// Search searches across all corpus types by keyword in transcript or tags.
func (d *DB) Search(query string) ([]*Item, error) {
	pattern := "%" + query + "%"
	rows, err := d.db.Query(`
		SELECT id, type, source, created_at, ingested_at, path, transcript, tags, metadata, word_count, duration_seconds, file_size
		FROM corpus_items
		WHERE transcript LIKE ? OR tags LIKE ? OR path LIKE ?
		ORDER BY ingested_at DESC`,
		pattern, pattern, pattern)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanItems(rows)
}

// TypeStats holds statistics for a single corpus type.
type TypeStats struct {
	Type       string
	Count      int
	TotalWords int
	TotalSize  int64
	TotalDur   float64
}

// Stats returns corpus statistics grouped by type.
func (d *DB) Stats() ([]TypeStats, error) {
	rows, err := d.db.Query(`
		SELECT type, COUNT(*), COALESCE(SUM(word_count), 0), COALESCE(SUM(file_size), 0), COALESCE(SUM(duration_seconds), 0)
		FROM corpus_items
		GROUP BY type
		ORDER BY type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var stats []TypeStats
	for rows.Next() {
		var s TypeStats
		if err := rows.Scan(&s.Type, &s.Count, &s.TotalWords, &s.TotalSize, &s.TotalDur); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, rows.Err()
}

// AllTranscripts returns all non-empty transcripts, optionally filtered by type.
func (d *DB) AllTranscripts(itemType string) ([]string, error) {
	var rows *sql.Rows
	var err error
	if itemType != "" {
		rows, err = d.db.Query(`SELECT transcript FROM corpus_items WHERE type = ? AND transcript != '' ORDER BY created_at`, itemType)
	} else {
		rows, err = d.db.Query(`SELECT transcript FROM corpus_items WHERE transcript != '' ORDER BY created_at`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transcripts []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		transcripts = append(transcripts, t)
	}
	return transcripts, rows.Err()
}

// Count returns total items, optionally filtered by type.
func (d *DB) Count(itemType string) (int, error) {
	var count int
	var err error
	if itemType != "" {
		err = d.db.QueryRow(`SELECT COUNT(*) FROM corpus_items WHERE type = ?`, itemType).Scan(&count)
	} else {
		err = d.db.QueryRow(`SELECT COUNT(*) FROM corpus_items`).Scan(&count)
	}
	return count, err
}

// ExportAll returns all corpus items.
func (d *DB) ExportAll() ([]*Item, error) {
	rows, err := d.db.Query(`SELECT id, type, source, created_at, ingested_at, path, transcript, tags, metadata, word_count, duration_seconds, file_size FROM corpus_items ORDER BY type, created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanItems(rows)
}

// scanner is implemented by both *sql.Row and *sql.Rows
type scanner interface {
	Scan(dest ...any) error
}

func scanItem(s scanner) (*Item, error) {
	var item Item
	var tagsStr, metaStr sql.NullString
	var createdAt sql.NullString
	var fileSize sql.NullInt64
	var wordCount sql.NullInt64
	var durFloat sql.NullFloat64

	err := s.Scan(&item.ID, &item.Type, &item.Source, &createdAt, &item.IngestedAt,
		&item.Path, &item.Transcript, &tagsStr, &metaStr, &wordCount, &durFloat, &fileSize)
	if err != nil {
		return nil, err
	}

	if createdAt.Valid {
		item.CreatedAt = createdAt.String
	}
	if wordCount.Valid {
		item.WordCount = int(wordCount.Int64)
	}
	if durFloat.Valid {
		item.DurationSeconds = durFloat.Float64
	}
	if fileSize.Valid {
		item.FileSize = fileSize.Int64
	}

	if tagsStr.Valid && tagsStr.String != "" {
		json.Unmarshal([]byte(tagsStr.String), &item.Tags)
	}
	if metaStr.Valid && metaStr.String != "" {
		json.Unmarshal([]byte(metaStr.String), &item.Metadata)
	}

	return &item, nil
}

func scanItems(rows *sql.Rows) ([]*Item, error) {
	var items []*Item
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// MigrateExistingCorpus scans voice-corpus directories for .ogg/.wav/.txt triplets
// and indexes them in the SQLite database.
func MigrateExistingCorpus(paths []string, db *DB) (int, error) {
	migrated := 0
	for _, dir := range paths {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		// Find all .txt files and create entries
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
				continue
			}

			base := strings.TrimSuffix(e.Name(), ".txt")
			txtPath := filepath.Join(dir, e.Name())

			transcript, err := os.ReadFile(txtPath)
			if err != nil {
				continue
			}
			text := strings.TrimSpace(string(transcript))
			if text == "" {
				continue
			}

			// Check if already in DB
			if existing, _ := db.GetByID(base); existing != nil {
				continue
			}

			info, _ := e.Info()
			var fileSize int64
			// Try to get size from the audio file
			for _, ext := range []string{".ogg", ".wav"} {
				if ai, err := os.Stat(filepath.Join(dir, base+ext)); err == nil {
					fileSize = ai.Size()
					break
				}
			}

			relPath := filepath.Join(filepath.Base(dir), e.Name())

			item := &Item{
				ID:         base,
				Type:       "voice",
				Source:     "local",
				IngestedAt: time.Now().Format(time.RFC3339),
				Path:       relPath,
				Transcript: text,
				WordCount:  len(strings.Fields(text)),
				FileSize:   fileSize,
			}
			if info != nil {
				item.CreatedAt = info.ModTime().Format(time.RFC3339)
			}

			if err := db.Insert(item); err != nil {
				continue
			}
			migrated++
		}

		// Also check transcripts/ subdirectory
		transcriptsDir := filepath.Join(dir, "transcripts")
		subEntries, err := os.ReadDir(transcriptsDir)
		if err != nil {
			continue
		}
		for _, e := range subEntries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".txt") {
				continue
			}
			base := strings.TrimSuffix(e.Name(), ".txt")
			txtPath := filepath.Join(transcriptsDir, e.Name())

			transcript, err := os.ReadFile(txtPath)
			if err != nil {
				continue
			}
			text := strings.TrimSpace(string(transcript))
			if text == "" {
				continue
			}

			if existing, _ := db.GetByID(base); existing != nil {
				continue
			}

			info, _ := e.Info()
			relPath := filepath.Join(filepath.Base(dir), "transcripts", e.Name())

			item := &Item{
				ID:         base,
				Type:       "voice",
				Source:     "local",
				IngestedAt: time.Now().Format(time.RFC3339),
				Path:       relPath,
				Transcript: text,
				WordCount:  len(strings.Fields(text)),
			}
			if info != nil {
				item.CreatedAt = info.ModTime().Format(time.RFC3339)
			}

			if err := db.Insert(item); err != nil {
				continue
			}
			migrated++
		}
	}
	return migrated, nil
}
