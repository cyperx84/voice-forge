# TASK: V3 Completion — Make It Production-Ready

## Context

V3 code exists but hasn't been fully tested against the real corpus.
Fix issues and make all V3 commands work end-to-end.

## What needs testing/fixing

### 1. `forge watch` — auto-capture
- Test with the real voice-corpus directory (~/.openclaw/workspace/voice-corpus/)
- The whisper transcription was broken last session (protobuf lib missing, no model installed)
- Fix: try OpenAI Whisper API as fallback if local whisper isn't available
- Add `--dry-run` flag to show what would be processed without actually doing it
- Make sure it handles .ogg files that already have .txt transcripts (skip them)
- Test fsnotify on macOS — make sure it detects new files

### 2. `forge refresh` — smart re-analysis
- Test the threshold logic (10% growth or 20+ new transcripts)
- Make sure it reads the existing style.json to compare sample counts
- Test `--force` flag
- Verify the new profile overwrites cleanly

### 3. `forge skill` — OpenClaw skill generation
- Test output generation to a temp dir first
- Compare generated SKILL.md quality with the hand-maintained one
- Generated skill should reference the forge profile as ground truth
- Test with `--output` pointing at ~/.openclaw/skills/cyperx-voice/

### 4. `forge pipeline` — full end-to-end
- Test: `forge pipeline` should run preprocess → score → export in sequence
- Verify it handles the real corpus dirs properly
- Check that scored files actually land in the export dir

### 5. Whisper fallback
The watcher and any transcription need a working whisper setup.
Priority order:
1. `mlx_whisper` (Apple Silicon native, fastest)
2. `whisper-cli` / `whisper-cpp` (local C++ build)
3. OpenAI Whisper API (`curl` to api.openai.com/v1/audio/transcriptions)

Add config option:
```toml
[watch]
whisper_command = "mlx_whisper"  # or "whisper-cli" or "openai-api"
whisper_model = "large-v3"
openai_api_key = ""             # for API fallback
```

### 6. Fix any compilation issues
- Run `go test ./...` and fix all failures
- Run `go vet ./...`
- Make sure `go build` produces a clean binary

## Success criteria
1. `forge watch --dry-run` shows unprocessed files in voice-corpus
2. `forge refresh --force` regenerates style.json from current corpus
3. `forge skill --output /tmp/test-skill/` produces valid SKILL.md
4. `forge pipeline` runs without errors on real corpus
5. All tests pass: `go test ./...`
6. Binary builds clean: `go build -o forge .`
