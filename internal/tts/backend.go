package tts

import (
	"fmt"
	"sort"
	"sync"
)

// AudioFormat describes the native output format of a TTS backend.
type AudioFormat struct {
	SampleRate int    // e.g., 24000
	Channels   int    // e.g., 1
	Codec      string // e.g., "pcm_f32le", "pcm_s16le", "mp3"
	Container  string // e.g., "wav", "mp3"
}

// Backend is the interface all TTS providers implement.
type Backend interface {
	Name() string
	Speak(text string, opts SpeakOpts) ([]byte, error) // returns WAV/MP3 audio bytes
	Clone(samples []string, name string) error          // voice cloning from audio samples
	Available() bool                                     // check if backend is ready
	Setup() error                                        // install/configure backend
	NativeFormat() AudioFormat                           // returns the backend's native output format
}

// SpeakOpts configures a single TTS request.
type SpeakOpts struct {
	Voice          string  // voice/model name
	Speed          float64 // speech rate multiplier
	OutputPath     string  // where to write audio file
	Format         string  // "wav" or "mp3"
	ReferenceAudio string  // path to reference audio for zero-shot cloning
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

// Names returns all registered backend names in sorted order.
func Names() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registry))
	for n := range registry {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// All returns all registered backends.
func All() []Backend {
	registryMu.RLock()
	defer registryMu.RUnlock()
	backends := make([]Backend, 0, len(registry))
	for _, b := range registry {
		backends = append(backends, b)
	}
	return backends
}

// ClearRegistry removes all registered backends (for testing).
func ClearRegistry() {
	registryMu.Lock()
	defer registryMu.Unlock()
	registry = map[string]Backend{}
}
