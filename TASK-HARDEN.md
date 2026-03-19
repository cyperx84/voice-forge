# Voice Forge — Production Hardening

## Context
Repo: `~/github/voice-forge`, Go + Cobra. Read `AUDIT.md` for full details. Fix ALL critical and important issues.

## Critical Fixes (must fix)

### 1. Config file permissions (config.go)
- `Save()` writes with `0644` — change to `0600` (owner-only read/write)
- API keys in config must not be world-readable

### 2. Race condition in continuous pipeline (watch.go / pipeline)
- `runContinuousPipeline()` and `w.Run()`'s ticker can both call `ProcessExisting()` concurrently
- Add a mutex to guard concurrent processing
- Ensure no double-processing of the same `.ogg` file

### 3. HTTP timeout (tts/elevenlabs.go)
- Replace `http.DefaultClient` with a client that has a 30s timeout
- `&http.Client{Timeout: 30 * time.Second}`

### 4. Subprocess timeouts (analyzer, rewriter, watch)
- Every `exec.Command` call must use `exec.CommandContext` with a timeout
- Analyzer (claude calls): 120s timeout
- Watch (whisper-cli): 60s timeout  
- Watch (ffmpeg): 30s timeout
- Rewriter (claude calls): 120s timeout

### 5. Division by zero (refresh logic)
- `needsRefresh()` divides by `existing.SampleCount`
- Guard against 0: if SampleCount is 0, always refresh

## Important Fixes

### 6. Input validation on `speak` command
- Validate text is not empty
- Validate backend exists before attempting TTS
- Validate output path is writable

### 7. Corpus size cap before LLM
- When piping corpus to LLM for analysis, cap at a reasonable size (e.g., 50K tokens worth of text)
- Truncate with a warning, don't just fail

### 8. Atomic file copy in ingest
- Use write-to-temp-then-rename pattern for file copies
- Prevents corrupted files if process is killed mid-copy

### 9. Fix TOML formatting in clone
- If there's manual TOML string construction, replace with proper TOML marshaling
- Use `github.com/pelletier/go-toml/v2` or `github.com/BurntSushi/toml`

### 10. TTS backend re-registration
- Prevent duplicate registration of backends
- Check if backend already registered before adding

### 11. Test coverage improvements
- Add tests for ALL the fixes above
- Target: cmd package from 10% to 40%+, tts package from 14% to 40%+
- Add edge case tests: empty corpus, missing config, invalid input, concurrent access

## Git
- Use `--no-gpg-sign` for all commits
- ONE commit: "fix: production hardening — security, race conditions, timeouts, validation"
- Push to `origin main` when done
- Run `go test ./...` and confirm all pass before committing

## When done
Run: `openclaw system event --text 'Done: Voice Forge hardening — all critical+important bugs fixed, tests added, pushed' --mode now`
