package tts

import (
	"fmt"
	"sync"
)

// Backend is the interface all TTS providers implement.
type Backend interface {
	Name() string
	Speak(text string, opts SpeakOpts) ([]byte, error) // returns WAV/MP3 audio bytes
	Clone(samples []string, name string) error          // voice cloning from audio samples
	Available() bool                                     // check if backend is ready
	Setup() error                                        // install/configure backend
}

// SpeakOpts configures a single TTS request.
type SpeakOpts struct {
	Voice      string  // voice/model name
	Speed      float64 // speech rate multiplier
	OutputPath string  // where to write audio file
	Format     string  // "wav" or "mp3"
}

var (
	registryMu sync.RWMutex
	registry   = map[string]Backend{}
)

// Register adds a backend to the global registry.
// If a backend with the same name is already registered, it is not replaced.
func Register(b Backend) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[b.Name()]; exists {
		return
	}
	registry[b.Name()] = b
}

// Get returns a registered backend by name.
func Get(name string) (Backend, error) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	b, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown TTS backend %q — available: %v", name, Names())
	}
	return b, nil
}

// Names returns all registered backend names.
func Names() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	return names
}
