package tts

import "fmt"

// KokoroBackend attempts direct Kokoro integration, falling back to tts-toolkit.
type KokoroBackend struct {
	Toolkit *ToolkitBackend // fallback
}

func (k *KokoroBackend) Name() string { return "kokoro" }

func (k *KokoroBackend) Available() bool {
	// No direct Go bindings exist yet — fall back to tts-toolkit
	return k.Toolkit != nil && k.Toolkit.Available()
}

func (k *KokoroBackend) Setup() error {
	if k.Available() {
		return nil
	}
	return fmt.Errorf("kokoro backend requires tts-toolkit — configure [tts.tts_toolkit] in ~/.forge/config.toml")
}

func (k *KokoroBackend) Speak(text string, opts SpeakOpts) ([]byte, error) {
	if !k.Available() {
		return nil, k.Setup()
	}
	// Force kokoro model via tts-toolkit
	opts.Voice = "kokoro"
	return k.Toolkit.Speak(text, opts)
}

func (k *KokoroBackend) Clone(samples []string, name string) error {
	if !k.Available() {
		return k.Setup()
	}
	return k.Toolkit.Clone(samples, name)
}
