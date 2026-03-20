# Voice Forge V5 — Working TTS + Bulk Ingest + Brew Tap

## Priority 1: Fix TTS Backends (forge speak must work end-to-end)

The current TTS backends all fail in practice:
- `tts-toolkit` backend: Python deps broken (missing soundfile module)
- `elevenlabs` backend: Needs API key (works architecturally but untested end-to-end)
- `kokoro` backend: Falls back to broken tts-toolkit

### Add Chatterbox Backend (NEW — TOP PRIORITY)
Research says Chatterbox Turbo is #1 for voice cloning (MIT, 350M params, beat ElevenLabs 63.75% in blind tests).

Add `internal/tts/chatterbox.go`:
- Calls Python directly via subprocess (not through tts-toolkit):
  ```
  python3 -c "
  from chatterbox.tts_turbo import ChatterboxTurboTTS
  tts = ChatterboxTurboTTS.from_pretrained()
  wav = tts.generate(text='...', audio_prompt_path='reference.wav')
  torchaudio.save('output.wav', wav.cpu(), 24000)
  "
  ```
- SpeakOpts should support reference_audio field for zero-shot cloning
- Available() checks: `python3 -c "import chatterbox"` succeeds
- Setup() prints install instructions: `pip3 install chatterbox-tts`
- Clone() saves reference audio clips to ~/.forge/voices/{name}/

### Add F5-TTS Backend (secondary)
Add `internal/tts/f5.go`:
- Similar subprocess approach
- Available() checks: `python3 -c "import f5_tts"` succeeds
- Zero-shot from reference audio

### Fix tts-toolkit Backend
- Make it resilient: if tts-toolkit import fails, return clear error with install instructions
- Don't crash, just report unavailable

### Add forge backends command
New command that lists all backends with availability status:
```
forge backends
  chatterbox    ✅ available (Chatterbox Turbo 350M)
  f5-tts        ❌ not installed (pip3 install f5-tts)
  elevenlabs    ❌ no API key (set in ~/.forge/config.toml)
  kokoro        ✅ available (via tts-toolkit)
  tts-toolkit   ❌ missing deps (pip3 install soundfile)
```

### Update config.toml
Add `[tts.chatterbox]` and `[tts.f5]` sections.
Default backend priority: chatterbox > f5 > kokoro > elevenlabs > tts-toolkit

## Priority 2: Bulk Ingest Commands

### forge ingest-bulk
New command that ingests everything at once:
```
forge ingest-bulk --voice ~/.openclaw/workspace/voice-corpus/ ~/.openclaw/workspace/voice/
forge ingest-bulk --code ~/github/voice-forge/ ~/github/tts-toolkit/
forge ingest-bulk --text ~/path/to/markdown/files/
```

Flags:
- `--voice <dirs...>` — ingest all .ogg/.wav/.txt triplets
- `--code <dirs...>` — ingest all .go/.py/.ts/.js/.md files (skip vendor/node_modules/.git)
- `--text <dirs...>` — ingest all .md/.txt files
- `--all` — run all configured paths from config.toml
- `--dry-run` — show what would be ingested without doing it

### forge corpus dedupe
Detect and remove duplicate entries (same file path or same content hash).

## Priority 3: Homebrew Formula

Create a Homebrew formula at `Formula/forge.go` (or just the formula file) for:
```
brew tap cyperx84/tap
brew install voice-forge
```

- Use GoReleaser-style approach
- Binary name: `forge`
- Dependencies: ffmpeg (required for preprocess/score)
- Optional deps: python3 (for TTS backends), whisper-cli (for transcription)

Create `homebrew-tap/Formula/voice-forge.rb` or output instructions for adding to existing tap at cyperx84/homebrew-tap.

## Priority 4: Polish

### Better error messages
- Every command that fails should suggest what to do
- Missing deps → print install command
- Missing config → print example config

### forge doctor
New command that checks entire environment:
```
forge doctor
  ✅ Config: ~/.forge/config.toml
  ✅ Corpus DB: ~/.forge/corpus.db (402 items)
  ✅ Style profile: ~/.forge/profile/style.json
  ✅ ffmpeg: installed (7.1)
  ❌ Chatterbox: not installed (pip3 install chatterbox-tts)
  ❌ Whisper: not installed
  ✅ ElevenLabs: API key configured
  ⚠️  Disk: corpus is 23.3MB
```

### Completion scripts
Ensure `forge completion bash/zsh/fish` works properly (Cobra provides this, just wire it up).

## Constraints
- Go only (no Python in the forge binary itself — subprocess calls for Python TTS)
- All tests must pass: `go test ./...`
- No GPG signing for git commits (use `git commit --no-gpg-sign`)
- Push to main branch on cyperx84/voice-forge
- Install binary: `go build -o forge . && cp forge ~/.local/bin/forge`

## After completing all work:
1. Run `go test ./...` and confirm all pass
2. Run `go build -o forge .` and confirm builds
3. Copy to `~/.local/bin/forge`
4. Git add, commit (--no-gpg-sign), push
5. Test: `forge doctor`, `forge backends`, `forge ingest-bulk --dry-run --all`
