package tts

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ToolkitBackend shells out to tts-toolkit for TTS.
type ToolkitBackend struct {
	Path         string // path to tts-toolkit directory
	DefaultModel string // default model (e.g. "kokoro")
}

func (t *ToolkitBackend) Name() string { return "tts-toolkit" }

func (t *ToolkitBackend) NativeFormat() AudioFormat {
	return AudioFormat{SampleRate: 24000, Channels: 1, Codec: "pcm_s16le", Container: "wav"}
}

func (t *ToolkitBackend) Available() bool {
	if t.Path == "" {
		return false
	}
	info, err := os.Stat(t.Path)
	return err == nil && info.IsDir()
}

func (t *ToolkitBackend) Setup() error {
	if t.Available() {
		return nil
	}
	return fmt.Errorf("tts-toolkit not found at %s — clone it from your tts-toolkit repo\n  install: git clone <your-tts-toolkit-repo> %s\n  also ensure Python deps: pip3 install soundfile numpy torch", t.Path, t.Path)
}

func (t *ToolkitBackend) Speak(text string, opts SpeakOpts) ([]byte, error) {
	if !t.Available() {
		return nil, fmt.Errorf("tts-toolkit not available at %s", t.Path)
	}

	model := opts.Voice
	if model == "" {
		model = t.DefaultModel
	}
	if model == "" {
		model = "kokoro"
	}

	format := opts.Format
	if format == "" {
		format = "wav"
	}

	outPath := opts.OutputPath
	if outPath == "" {
		tmp, err := os.CreateTemp("", "forge-speak-*."+format)
		if err != nil {
			return nil, fmt.Errorf("creating temp file: %w", err)
		}
		outPath = tmp.Name()
		tmp.Close()
		defer os.Remove(outPath)
	}

	args := []string{"speak", "--model", model, "--output", outPath}
	if opts.Speed != 0 {
		args = append(args, "--speed", fmt.Sprintf("%.2f", opts.Speed))
	}
	args = append(args, text)

	scriptPath := filepath.Join(t.Path, "tts-toolkit")
	cmd := exec.Command(scriptPath, args...)
	cmd.Dir = t.Path
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tts-toolkit failed: %w\noutput: %s", err, out)
	}

	audio, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("reading output audio: %w", err)
	}

	return audio, nil
}

func (t *ToolkitBackend) Clone(samples []string, name string) error {
	if !t.Available() {
		return fmt.Errorf("tts-toolkit not available at %s", t.Path)
	}

	args := []string{"clone", "--name", name}
	args = append(args, samples...)

	scriptPath := filepath.Join(t.Path, "tts-toolkit")
	cmd := exec.Command(scriptPath, args...)
	cmd.Dir = t.Path
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tts-toolkit clone failed: %w\noutput: %s", err, out)
	}
	return nil
}
