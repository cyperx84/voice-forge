# Voice Forge V1 — TTS Backend + forge speak

## Context
V0 is done: `forge analyze`, `forge stats`, `forge ingest`, `forge profile` all working. Style profile extracted at `~/.forge/profile/style.json`. Repo: `~/github/voice-forge`, Go + Cobra.

## What to Build

### 1. Pluggable TTS Backend Interface
Create `internal/tts/` package:

```go
// Backend is the interface all TTS providers implement
type Backend interface {
    Name() string
    Speak(text string, opts SpeakOpts) ([]byte, error)  // returns WAV/MP3 audio bytes
    Clone(samples []string, name string) error           // voice cloning from audio samples
    Available() bool                                      // check if backend is ready
    Setup() error                                         // install/configure backend
}

type SpeakOpts struct {
    Voice      string  // voice/model name
    Speed      float64 // speech rate multiplier
    OutputPath string  // where to write audio file
    Format     string  // "wav" or "mp3"
}
```

### 2. Backends to Implement

**Priority 1: tts-toolkit (shell out to Python)**
- Shell out to `~/github/tts-toolkit` CLI or Python module
- This already has 7 backends (Kokoro, Chatterbox, XTTS, Fish, etc.)
- Config: which tts-toolkit backend to use, model path
- `forge speak "text" --backend tts-toolkit --model kokoro`

**Priority 2: ElevenLabs API**
- REST API calls from Go (no Python dependency)
- Needs API key in `~/.forge/config.toml` under `[tts.elevenlabs]`
- Voice cloning: upload samples via API
- `forge speak "text" --backend elevenlabs --voice "CyperX"`

**Priority 3: Local Kokoro (direct)**
- Direct integration if kokoro-onnx or similar Go bindings exist
- Falls back to tts-toolkit wrapper if not

### 3. New Commands

**`forge speak "text"`**
- Takes text, generates audio in cloned voice
- Uses default backend from config, overridable with `--backend`
- Outputs to stdout (pipe-friendly) or `--output file.wav`
- Applies style hints from `~/.forge/profile/style.json` if `--style` flag set
- Example: `forge speak "let's rock and roll" --output test.wav`

**`forge clone`**
- `forge clone --provider tts-toolkit --model xtts` — clone voice using corpus samples
- Selects best N samples from corpus automatically (longest, clearest)
- Saves voice model/embedding to `~/.forge/voices/cyperx/`
- Config remembers default voice

**`forge voices`**
- List available cloned voices
- Show which backend each voice uses

### 4. Config Updates
Add to `~/.forge/config.toml`:
```toml
[tts]
default_backend = "tts-toolkit"

[tts.tts_toolkit]
path = "/Users/cyperx/github/tts-toolkit"
default_model = "kokoro"

[tts.elevenlabs]
api_key = ""  # or read from env ELEVENLABS_API_KEY

[voices]
default = "cyperx"
```

### 5. Tests
- Unit tests for TTS interface
- Integration test that mocks backend
- Test config loading with TTS section

### 6. Git
- Use `--no-gpg-sign` for all commits
- Push to `origin main` when done
- Small atomic commits

## Constraints
- Go only (no Python in this repo — shell out to tts-toolkit as external tool)
- All TTS backends are optional — `forge speak` should give helpful error if no backend configured
- Keep it simple — V1 just needs to work, V2 adds character modes
- Audio output: WAV preferred, MP3 optional
- Don't install Python packages or pip install anything — just shell out to tts-toolkit if it's there
