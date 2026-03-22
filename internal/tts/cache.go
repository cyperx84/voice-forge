package tts

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// Cache provides content-addressable caching for TTS audio.
type Cache struct {
	Dir string // cache directory (e.g., ~/.forge/cache/tts/)
}

// cacheKey generates a deterministic key from text, voice, and backend.
func cacheKey(text, voice, backend string) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s\x00%s\x00%s", text, voice, backend)))
	return hex.EncodeToString(h[:])
}

// Get retrieves cached audio bytes. Returns nil, false on cache miss.
func (c *Cache) Get(text, voice, backend string) ([]byte, bool) {
	if c.Dir == "" {
		return nil, false
	}
	key := cacheKey(text, voice, backend)
	path := filepath.Join(c.Dir, key+".wav")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

// Put stores audio bytes in the cache.
func (c *Cache) Put(text, voice, backend string, audio []byte) error {
	if c.Dir == "" {
		return nil
	}
	if err := os.MkdirAll(c.Dir, 0755); err != nil {
		return fmt.Errorf("creating cache dir: %w", err)
	}
	key := cacheKey(text, voice, backend)
	path := filepath.Join(c.Dir, key+".wav")
	return os.WriteFile(path, audio, 0644)
}
