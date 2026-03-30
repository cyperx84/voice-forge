package tts

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// F5Backend calls F5-TTS via Python subprocess for TTS.
type F5Backend struct {
	VoicesDir string // directory to store reference audio (~/.forge/voices/)
	Runtime   PythonRuntime
}

func (f *F5Backend) Name() string { return "f5-tts" }

func (f *F5Backend) NativeFormat() AudioFormat {
	return AudioFormat{SampleRate: 24000, Channels: 1, Codec: "pcm_f32le", Container: "wav"}
}

func (f *F5Backend) pythonBin() string {
	if f.Runtime.PythonPath != "" {
		return f.Runtime.PythonPath
	}
	return ResolveConfiguredRuntime("FORGE_F5_PYTHON", "", ".forge/venvs/f5-tts").PythonPath
}

func (f *F5Backend) Available() bool {
	cmd := exec.Command(f.pythonBin(), "-c", "import f5_tts")
	return cmd.Run() == nil
}

func (f *F5Backend) Setup() error {
	if f.Available() {
		return nil
	}
	return fmt.Errorf("f5-tts not installed — install with: python3 -m venv ~/.forge/venvs/f5-tts && source ~/.forge/venvs/f5-tts/bin/activate && pip install f5-tts")
}

func (f *F5Backend) Speak(text string, opts SpeakOpts) ([]byte, error) {
	if !f.Available() {
		return nil, f.Setup()
	}

	text = strings.ReplaceAll(text, "'", "\\'")
	text = strings.ReplaceAll(text, "\"", "\\\"")

	format := opts.Format
	if format == "" {
		format = "wav"
	}

	outPath := opts.OutputPath
	if outPath == "" {
		tmp, err := os.CreateTemp("", "forge-f5-*."+format)
		if err != nil {
			return nil, fmt.Errorf("creating temp file: %w", err)
		}
		outPath = tmp.Name()
		tmp.Close()
		defer os.Remove(outPath)
	}

	refAudio := opts.ReferenceAudio
	if refAudio == "" {
		refAudio = f.findVoiceRef(opts.Voice)
	}

	var script string
	if refAudio != "" {
		script = fmt.Sprintf(`
from f5_tts.api import F5TTS
tts = F5TTS()
tts.infer(
    ref_file='%s',
    gen_text='%s',
    file_wave='%s',
)
`, refAudio, text, outPath)
	} else {
		script = fmt.Sprintf(`
from f5_tts.api import F5TTS
tts = F5TTS()
tts.infer(
    gen_text='%s',
    file_wave='%s',
)
`, text, outPath)
	}

	cmd := exec.Command(f.pythonBin(), "-c", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("f5-tts failed: %w\noutput: %s", err, out)
	}

	audio, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("reading output audio: %w", err)
	}

	return audio, nil
}

func (f *F5Backend) Clone(samples []string, name string) error {
	if !f.Available() {
		return f.Setup()
	}

	voiceDir := filepath.Join(f.VoicesDir, name)
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

func (f *F5Backend) findVoiceRef(voice string) string {
	if voice == "" || f.VoicesDir == "" {
		return ""
	}
	voiceDir := filepath.Join(f.VoicesDir, voice)
	for _, ext := range []string{".wav", ".ogg", ".mp3", ".m4a"} {
		ref := filepath.Join(voiceDir, "reference_0"+ext)
		if _, err := os.Stat(ref); err == nil {
			return ref
		}
	}
	return ""
}
