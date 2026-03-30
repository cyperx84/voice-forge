---
name: voice-forge
description: Generate speech audio using Voice Forge TTS engine
---

# Voice Forge

Generate high-quality speech audio from text using configurable TTS backends with voice cloning support.

## Usage

```bash
forge speak "text to speak" --output path.wav [options]
```

### Options

| Flag | Description |
|------|-------------|
| `--output`, `-o` | Output file path (writes to stdout if omitted) |
| `--voice` | Voice/model name (default: from config) |
| `--backend` | TTS backend: `chatterbox`, `f5-tts`, `elevenlabs`, `tts-toolkit`, `kokoro` |
| `--preset` | Output format preset: `discord`, `podcast`, `video`, `lossless` |
| `--normalize` | Normalize to standard mixing format (44.1kHz pcm_s16le mono WAV) |
| `--speed` | Speech rate multiplier |
| `--character` | Character persona for tone shifting |
| `--discord` | Alias for `--preset discord` |
| `--listen-link` | Generate self-contained HTML player page |

### Output Presets

| Preset | Format | Sample Rate | Channels | Bitrate | Use Case |
|--------|--------|-------------|----------|---------|----------|
| `discord` | MP3 | 48kHz | Mono | 128k | Discord attachment player |
| `podcast` | MP3 | 44.1kHz | Stereo | 192k | Long-form audio content |
| `video` | AAC/M4A | 48kHz | Stereo | 192k | Video narration |
| `lossless` | WAV | 44.1kHz | Mono | — | Editing and mixing |

## Examples

### Generate a voice clip
```bash
forge speak "let's rock and roll" --output clip.wav --voice cyperx
```

### Discord-ready audio
```bash
forge speak "ship it" --preset discord --output message.mp3
```

### Podcast narration
```bash
forge speak "Welcome to the show" --preset podcast --output intro.mp3 --listen-link
```

### Multi-line script (parallel generation)
```bash
forge speak "Line one.
Line two.
Line three." --output script.wav
```

## Voice Discovery

Available voices depend on reference audio stored in `~/.forge/voices/`:
```
~/.forge/voices/
├── cyperx/
│   └── reference_0.wav
└── narrator/
    └── reference_0.wav
```

## Performance Notes

- **Chatterbox**: ~10s per line on CPU, faster on GPU/MPS. Multi-line input uses parallel generation (4 workers by default).
- **ElevenLabs**: Fast cloud API, requires API key.
- **F5-TTS**: resolves independently from Chatterbox and should use its own runtime path; still verify with `forge backends` before using it in automation.

## Output

Returns the file path to the generated audio file. When using `--preset`, the output is transcoded to the specified format via ffmpeg.

## Agent workflow

1. Run `forge backends` before assuming a backend is healthy.
2. Prefer `--preset discord` for Discord uploads.
3. Use `--listen-link` when you want a hostable single-file audio page.
4. If generation is long-running, post progress updates instead of going silent.

## Runtime isolation

- Chatterbox resolves `FORGE_CHATTERBOX_PYTHON`, then `[tts.chatterbox].runtime_path`, then `~/.forge/venvs/chatterbox/bin/python3`.
- F5 resolves `FORGE_F5_PYTHON`, then `[tts.f5].runtime_path`, then `~/.forge/venvs/f5-tts/bin/python3`.
- If you are automating setup on macOS, use `./scripts/setup-runtimes.sh` first, then verify with `forge backends`.
