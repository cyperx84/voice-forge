package tts

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ChatterboxBackend calls Chatterbox Turbo via Python subprocess for TTS.
type ChatterboxBackend struct {
	VoicesDir string // directory to store reference audio (~/.forge/voices/)
}

func (c *ChatterboxBackend) Name() string { return "chatterbox" }

func (c *ChatterboxBackend) Available() bool {
	cmd := exec.Command("python3", "-c", "import chatterbox")
	return cmd.Run() == nil
}

func (c *ChatterboxBackend) Setup() error {
	if c.Available() {
		return nil
	}
	return fmt.Errorf("chatterbox not installed — install with: pip3 install chatterbox-tts")
}

func (c *ChatterboxBackend) Speak(text string, opts SpeakOpts) ([]byte, error) {
	if !c.Available() {
		return nil, c.Setup()
	}

	text = strings.ReplaceAll(text, "'", "\\'")
	text = strings.ReplaceAll(text, "\"", "\\\"")

	format := opts.Format
	if format == "" {
		format = "wav"
	}

	outPath := opts.OutputPath
	if outPath == "" {
		tmp, err := os.CreateTemp("", "forge-chatterbox-*."+format)
		if err != nil {
			return nil, fmt.Errorf("creating temp file: %w", err)
		}
		outPath = tmp.Name()
		tmp.Close()
		defer os.Remove(outPath)
	}

	// Build Python script
	var script string
	if opts.ReferenceAudio != "" {
		script = fmt.Sprintf(`
import torchaudio
from chatterbox.tts_turbo import ChatterboxTurboTTS
tts = ChatterboxTurboTTS.from_pretrained()
wav = tts.generate(text='%s', audio_prompt_path='%s')
torchaudio.save('%s', wav.cpu(), 24000)
`, text, opts.ReferenceAudio, outPath)
	} else {
		// Check for default voice reference
		refAudio := c.findVoiceRef(opts.Voice)
		if refAudio != "" {
			script = fmt.Sprintf(`
import torchaudio
from chatterbox.tts_turbo import ChatterboxTurboTTS
tts = ChatterboxTurboTTS.from_pretrained()
wav = tts.generate(text='%s', audio_prompt_path='%s')
torchaudio.save('%s', wav.cpu(), 24000)
`, text, refAudio, outPath)
		} else {
			script = fmt.Sprintf(`
import torchaudio
from chatterbox.tts_turbo import ChatterboxTurboTTS
tts = ChatterboxTurboTTS.from_pretrained()
wav = tts.generate(text='%s')
torchaudio.save('%s', wav.cpu(), 24000)
`, text, outPath)
		}
	}

	cmd := exec.Command("python3", "-c", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("chatterbox failed: %w\noutput: %s", err, out)
	}

	audio, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("reading output audio: %w", err)
	}

	return audio, nil
}

func (c *ChatterboxBackend) Clone(samples []string, name string) error {
	if !c.Available() {
		return c.Setup()
	}

	voiceDir := filepath.Join(c.VoicesDir, name)
	if err := os.MkdirAll(voiceDir, 0755); err != nil {
		return fmt.Errorf("creating voice directory: %w", err)
	}

	for i, sample := range samples {
		data, err := os.ReadFile(sample)
		if err != nil {
			return fmt.Errorf("reading sample %s: %w", sample, err)
		}
		ext := filepath.Ext(sample)
		if ext == "" {
			ext = ".wav"
		}
		dest := filepath.Join(voiceDir, fmt.Sprintf("reference_%d%s", i, ext))
		if err := os.WriteFile(dest, data, 0644); err != nil {
			return fmt.Errorf("writing reference audio: %w", err)
		}
	}

	fmt.Printf("Saved %d reference clip(s) to %s\n", len(samples), voiceDir)
	return nil
}

// findVoiceRef looks for a reference audio file in the voices directory.
func (c *ChatterboxBackend) findVoiceRef(voice string) string {
	if voice == "" || c.VoicesDir == "" {
		return ""
	}
	voiceDir := filepath.Join(c.VoicesDir, voice)
	for _, ext := range []string{".wav", ".ogg", ".mp3", ".m4a"} {
		ref := filepath.Join(voiceDir, "reference_0"+ext)
		if _, err := os.Stat(ref); err == nil {
			return ref
		}
	}
	return ""
}
