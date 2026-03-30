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
	Runtime   PythonRuntime
}

func (c *ChatterboxBackend) Name() string { return "chatterbox" }

func (c *ChatterboxBackend) NativeFormat() AudioFormat {
	return AudioFormat{SampleRate: 24000, Channels: 1, Codec: "pcm_f32le", Container: "wav"}
}

func (c *ChatterboxBackend) pythonBin() string {
	if c.Runtime.PythonPath != "" {
		return c.Runtime.PythonPath
	}
	return ResolveConfiguredRuntime("FORGE_CHATTERBOX_PYTHON", "", ".forge/venvs/chatterbox").PythonPath
}

func (c *ChatterboxBackend) Available() bool {
	cmd := exec.Command(c.pythonBin(), "-c", "import chatterbox")
	return cmd.Run() == nil
}

func (c *ChatterboxBackend) Setup() error {
	if c.Available() {
		return nil
	}
	return fmt.Errorf("chatterbox not installed — install with: python3 -m venv ~/.forge/venvs/chatterbox && source ~/.forge/venvs/chatterbox/bin/activate && pip install chatterbox-tts")
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

	// Resolve reference audio
	refAudio := opts.ReferenceAudio
	if refAudio == "" {
		refAudio = c.findVoiceRef(opts.Voice)
	}

	// Build Python script — monkey-patch perth (native watermarker doesn't build on ARM/py3.13),
	// then load from cached weights directly to avoid HF auth
	preamble := `
import perth
if perth.PerthImplicitWatermarker is None:
    perth.PerthImplicitWatermarker = perth.DummyWatermarker
import os, glob, torch, torchaudio
from chatterbox.tts import ChatterboxTTS

device = 'cpu'
if torch.backends.mps.is_available():
    device = 'mps'
elif torch.cuda.is_available():
    device = 'cuda'

# Find cached model weights
cache_dir = os.path.expanduser('~/.cache/huggingface/hub/models--ResembleAI--chatterbox/snapshots')
snapshots = sorted(glob.glob(os.path.join(cache_dir, '*')))
if not snapshots:
    raise RuntimeError(f'No cached chatterbox model found in {cache_dir}. Run: forge doctor')
model_path = snapshots[-1]

tts = ChatterboxTTS.from_local(model_path, device)
`

	var script string
	if refAudio != "" {
		script = fmt.Sprintf(`%s
wav = tts.generate(text='%s', audio_prompt_path='%s')
torchaudio.save('%s', wav.cpu(), 24000)
`, preamble, text, refAudio, outPath)
	} else {
		script = fmt.Sprintf(`%s
wav = tts.generate(text='%s')
torchaudio.save('%s', wav.cpu(), 24000)
`, preamble, text, outPath)
	}

	cmd := exec.Command(c.pythonBin(), "-c", script)
	// Use cached model weights — don't hit HF API on every call
	cmd.Env = append(os.Environ(), "HF_HUB_OFFLINE=1", "TOKENIZERS_PARALLELISM=false")
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
