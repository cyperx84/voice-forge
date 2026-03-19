# Voice Forge — Production Readiness Audit

**Date:** 2026-03-19  
**Audited by:** Builder (subagent)  
**Coverage baseline:** `go test ./... -cover`

| Package | Coverage |
|---------|----------|
| `main` | 0.0% |
| `cmd` | 10.1% |
| `internal/analyzer` | 67.4% |
| `internal/character` | 72.7% |
| `internal/config` | 63.2% |
| `internal/corpus` | 92.7% |
| `internal/profile` | 96.4% |
| `internal/rewriter` | 94.4% |
| `internal/skill` | 95.2% |
| `internal/tts` | 14.4% |
| `internal/watch` | 32.7% |

---

## Critical

### 1. API Key Stored World-Readable — `internal/config/config.go:Save()`

The ElevenLabs API key is stored in `~/.forge/config.toml` as plaintext. `Save()` writes with mode `0644`, making it readable by every user on the machine.

```go
// config.go:133
return os.WriteFile(path, data, 0644)  // ← should be 0600
```

**Fix:** Change to `0600`. Also consider reading the key exclusively from an env var (`ELEVENLABS_API_KEY`) instead of storing it in config at all, or at minimum document that the file must be `chmod 600`.

---

### 2. Race Condition: Concurrent `ProcessExisting()` Calls — `cmd/pipeline.go` + `internal/watch/watch.go`

`runContinuousPipeline()` spawns a goroutine that periodically calls `runOncePipeline()` (which calls `w.ProcessExisting()`), while `w.Run(stop)` also calls `w.ProcessExisting()` on its poll ticker in the main goroutine. Both can fire simultaneously, causing double-processing of the same `.ogg` file.

```go
// pipeline.go:~100 (goroutine)
case <-ticker.C:
    runOncePipeline(cfg, skillOutput)  // calls w.ProcessExisting()

// watch.go:Run() (main goroutine, same Watcher)
case <-ticker.C:
    w.ProcessExisting()  // concurrent with above
```

The `Watcher` struct has no mutex protecting its state. Two concurrent `ingest()` calls for the same file will both pass the `hasTranscript()` check (the transcript file doesn't exist yet), then both invoke ffmpeg and whisper on it. The second call's ffmpeg will fail (WAV already exists), but both whisper calls will run.

**Fix:** Add a `sync.Map` or mutex-protected set of in-progress files in `Watcher`. Or deduplicate by having the pipeline's periodic refresh skip the `ProcessExisting` call and rely solely on `w.Run()`'s own ticker.

---

### 3. No HTTP Timeout on ElevenLabs Client — `internal/tts/elevenlabs.go`

Both `Speak()` and `Clone()` use `http.DefaultClient` which has no timeout. A hung API connection blocks the goroutine forever.

```go
// elevenlabs.go:52
resp, err := http.DefaultClient.Do(req)
```

**Fix:**
```go
client := &http.Client{Timeout: 120 * time.Second}
resp, err := client.Do(req)
```

---

### 4. No Context / Timeout for Subprocess Calls

`exec.Command` is used without a context in three places. A hanging `claude`, `whisper-cli`, or `ffmpeg` process will block the parent process indefinitely.

- `internal/analyzer/analyzer.go:100` — LLM analysis call
- `internal/rewriter/rewriter.go:runLLM()` — LLM rewrite call  
- `internal/watch/watch.go:ingest()` — ffmpeg and whisper calls

**Fix:** Use `exec.CommandContext(ctx, ...)` with a configurable timeout. For `watch`, wrap in a context with deadline:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
defer cancel()
cmd := exec.CommandContext(ctx, w.WhisperCommand, args...)
```

---

### 5. Division by Zero in `needsRefresh` — `cmd/refresh.go:~80`

If `existing.SampleCount` is 0 (e.g., a profile was manually created or seeded with `sample_count: 0`), the growth percentage calculation panics.

```go
growthPct := float64(newTranscripts) / float64(existing.SampleCount) * 100  // ← divide by zero if 0
```

**Fix:**
```go
var growthPct float64
if existing.SampleCount > 0 {
    growthPct = float64(newTranscripts) / float64(existing.SampleCount) * 100
}
```

---

## Important

### 6. No Input Validation on `forge speak` Text — `cmd/speak.go`

Empty string input is passed directly to the TTS backend with no validation. Very long strings (hundreds of KB) are also passed unchecked — this could cause ElevenLabs API errors, exceed tts-toolkit limits, or send arbitrarily large prompts to the rewriter LLM.

```go
// speak.go:RunE
text := args[0]   // no length or content validation
```

**Fix:** Add minimum/maximum length checks. For ElevenLabs specifically, enforce the API's 5,000-character limit. For the rewriter path, warn if the text is too long to rewrite effectively.

---

### 7. No Size Limit on LLM Corpus Input — `internal/analyzer/analyzer.go:Analyze()`

All transcripts are joined into a single string and sent as one prompt with no size cap. A corpus of thousands of recordings could produce a multi-megabyte prompt, exceeding model context limits and causing expensive failures late in a long-running operation.

```go
corpus := strings.Join(transcripts, "\n\n---\n\n")
prompt := fmt.Sprintf(analysisPrompt, corpus)
```

**Fix:** Add a configurable `MaxTranscripts` or `MaxWords` config option. Sample the corpus if it exceeds the limit (e.g., random or most-recent selection). Log a warning when truncation occurs.

---

### 8. Non-Atomic File Copy in `cmd/ingest.go:copyFile()`

The copy is done with `os.Create()` + `io.Copy()`. If the process is killed mid-copy, a partial file is left at the destination. On next run, the file will be skipped (`os.Stat(dest) == nil`) but will be corrupt.

Also, the destination file handle is not synced before close, risking data loss on crash.

```go
// ingest.go:copyFile
out, err := os.Create(dst)
defer out.Close()
_, err = io.Copy(out, in)
return err  // no out.Sync() call
```

**Fix:** Write to a temp file first, then `os.Rename()` (atomic on Linux/macOS). Add `out.Sync()` before returning.

---

### 9. Manual TOML Construction — `cmd/clone.go:~100`

Voice metadata is written as a hand-formatted string instead of using the TOML marshaler. Names with special characters (quotes, backslashes) would produce invalid TOML.

```go
meta := fmt.Sprintf("name = %q\nbackend = %q\nsamples = %d\n", name, backendName, len(samples))
```

**Fix:** Define a `VoiceMeta` struct and marshal it with `toml.Marshal()`.

---

### 10. `initBackends()` Re-registers on Every Command — `cmd/speak.go`

`initBackends()` calls `tts.Register()` without checking if a backend is already registered. Multiple invocations (e.g., in integration tests or if `PersistentPreRunE` triggers again) silently overwrite existing registrations. The global registry is also never cleared between invocations.

**Fix:** Either make `Register()` idempotent (return error on duplicate), or expose a `Clear()` function for tests, or use a per-command local registry rather than a global.

---

### 11. `isStopWord()` Allocates a New Map on Every Call — `internal/corpus/corpus.go`

The stop words map is re-created on every call to `isStopWord()`. For large corpora with millions of words, this causes significant unnecessary GC pressure.

```go
func isStopWord(w string) bool {
    stops := map[string]bool{   // ← allocated fresh on every call
        "the": true, ...
    }
    return stops[w]
}
```

**Fix:** Move `stops` to a package-level `var`.

---

### 12. `forge watch` Hardcoded 500ms File-Write Delay — `internal/watch/watch.go:91`

```go
time.Sleep(500 * time.Millisecond)  // ← magic number
```

This is fragile: too short for slow NFS mounts or large files, unnecessary overhead for fast local SSDs. Should be configurable via `WatchConfig`.

---

### 13. `tts/backend.go:Names()` Returns Non-Deterministic Order

`Names()` iterates over the `registry` map, whose order is randomized per Go spec. This produces inconsistent error messages ("available: [elevenlabs tts-toolkit]" vs "available: [tts-toolkit elevenlabs]").

**Fix:** Sort the names slice before returning it.

---

### 14. No Validation of Audio File Type Before ElevenLabs Upload — `internal/tts/elevenlabs.go:Clone()`

`Clone()` opens any file path provided in `samples` and uploads it to ElevenLabs. If a non-audio file (e.g., a `.txt` transcript) sneaks into the samples list, it's uploaded as audio without validation.

**Fix:** Check file extension against allowed audio types before uploading. Consider checking magic bytes for common audio formats.

---

### 15. Low Test Coverage in Critical Areas

**`cmd` package (10.1%):** All 9 command handlers (`analyze`, `clone`, `ingest`, `pipeline`, `speak`, `watch`, `write`, `refresh`, `character`) have essentially no test coverage. Command flag parsing, config loading paths, and backend selection logic are entirely untested.

**`internal/tts` (14.4%):** The real backends (`ToolkitBackend.Speak`, `ElevenLabsBackend.Speak`, `ElevenLabsBackend.Clone`, `KokoroBackend.Speak`) have 0% coverage. The mock is used for registry tests only — it doesn't simulate error conditions, timeout behavior, or HTTP errors.

**`internal/watch` (32.7%):** `Run()` (the core loop with fsnotify), `ingest()`, and `transcribe()` are untested. Only helper functions are covered.

**`internal/analyzer` (67.4%):** The `Analyze()` function itself (the LLM call + JSON parsing) is not tested. The JSON fallback extraction path (`strings.Index` hack) is untested.

**Specific gaps:**
- No test for `needsRefresh` when `SampleCount == 0`
- No test for concurrent `ProcessExisting()` calls (the race from finding #2)
- No test for `ElevenLabs` HTTP error responses
- No test for `stripCodeFences` with mixed fence formats
- No integration test for full pipeline flow
- `TestSaveAndLoad` in `config_test.go` is effectively a no-op (skips if `~/.forge` doesn't exist)

---

### 16. `config.go:ExpandPath()` Silently Falls Back on `os.UserHomeDir()` Error

```go
func ExpandPath(path string) string {
    if strings.HasPrefix(path, "~/") {
        home, err := os.UserHomeDir()
        if err != nil {
            return path  // ← silently returns unexpanded path
        }
        ...
    }
}
```

If `UserHomeDir()` fails (container, broken passwd), every path that starts with `~/` is returned unexpanded. The caller then tries to use `~/foo/bar` as a literal path, producing confusing "no such file" errors with no indication of the root cause.

**Fix:** Return an error from `ExpandPath()` or at minimum `log.Printf` the failure before falling back.

---

## Nice-to-have

### 17. Hardcoded "cyperx" References Throughout

Multiple places have the user's identity hardcoded:
- `internal/character/character.go:builtinPresets` — all preset characters use `BasedOn: "cyperx"`
- `internal/skill/skill.go:generateSkillMd()` — hardcodes `name: cyperx-voice` in YAML frontmatter and "CyperX" in prose
- `cmd/clone.go:~60` — fallback `name = "cyperx"`
- `config.go:DefaultConfig()` — `Voices.Default = "cyperx"`

For an open-source tool, these should be configurable via the config file.

---

### 18. `cmd/speak.go` Silently Falls Back on Style Profile Load Failure

```go
styleJSON, err := rewriter.LoadStyleJSON(stylePath)
if err != nil {
    styleJSON = "{}"   // ← silent fallback, no warning
}
```

The rewriter will then operate with an empty style profile, producing poor-quality output. This should at least log a warning with the path and error.

---

### 19. `cmd/pipeline.go:runContinuousPipeline()` — Signal Handler Leak

`signal.Notify(sig, ...)` is called but `signal.Stop(sig)` is never called when the function returns. Goroutines listening on `sig` may not be cleaned up properly on abnormal exit. Same issue exists in `cmd/watch.go`.

**Fix:** `defer signal.Stop(sig)` immediately after `signal.Notify(sig, ...)`.

---

### 20. `internal/corpus/corpus.go:ReadTranscripts()` — Errors Silently Swallowed

Individual file read errors are silently `continue`d:

```go
data, err := os.ReadFile(f)
if err != nil {
    continue   // ← no logging, caller never knows
}
```

In a production pipeline, a permissions error on one file silently reduces the corpus quality. Should at least log a warning.

---

### 21. No `go.sum` Integrity Check in CI / No `go vet` / No Linter Config

There's no `.github/workflows/`, no `Makefile`, and no linter configuration (`.golangci.yml`). Running `go vet ./...` locally produces no issues, but without enforced CI, regressions won't be caught automatically.

---

### 22. `ElevenLabsBackend` Has No Rate Limiting or Retry Logic

ElevenLabs enforces rate limits. Receiving a `429 Too Many Requests` returns a generic error with no retry. For automated pipeline use, this is a reliability gap.

---

### 23. `forge write` and `forge voices` and `forge stats` — Missing from Audit Trail

The files `cmd/write.go`, `cmd/voices.go`, `cmd/stats.go`, `cmd/skill.go`, `cmd/profile.go` were not read. If they contain similar patterns (subprocess calls, file writes, LLM usage), they likely have the same categories of issues. A follow-up review is recommended.

---

## Architecture Assessment

**Strengths:**
- The `tts.Backend` interface is well-designed. Adding a new TTS provider requires implementing four methods and calling `tts.Register()`. Clean.
- Config system is solid: TOML with sane defaults, `ExpandPath()` for `~` handling, and per-method accessors on `Config`.
- Package boundaries are logical: `corpus`, `analyzer`, `rewriter`, `character`, `skill` are appropriately separated.
- The `Watch` + `ProcessExisting` dual-path (fsnotify + poll ticker) is a good reliability pattern for NFS/remote mounts.
- The `Analyze()` LLM prompt is structured and produces consistent JSON output with a multi-stage JSON-extraction fallback.

**Weaknesses:**
- The global TTS backend registry (`internal/tts/backend.go`) is a singleton that's never reset between commands. This makes the package hard to test in isolation and creates subtle re-registration issues.
- `cmd/` commands share mutable package-level flag variables (`speakBackend`, `speakVoice`, etc.) which is idiomatic Cobra but means commands are not safe to call concurrently from tests.
- The `runContinuousPipeline`/`w.Run` relationship creates an implicit coupling: the pipeline owns a `Watcher` that also independently polls — the two ticker loops process the same directory without coordination.
- No structured logging (everything uses `log.Printf`). For a tool designed to run as a cron/daemon, structured logs (e.g., `slog`) would make filtering and monitoring much easier.
- No observability hooks: no metrics, no way to monitor pipeline throughput or error rates from outside the process.

---

## Summary Priority Matrix

| # | Finding | Severity | Effort |
|---|---------|----------|--------|
| 1 | API key world-readable (0644) | Critical | Low |
| 2 | Race condition in concurrent ProcessExisting | Critical | Medium |
| 3 | No HTTP timeout (ElevenLabs) | Critical | Low |
| 4 | No subprocess timeout (LLM/ffmpeg/whisper) | Critical | Low |
| 5 | Division by zero in needsRefresh | Critical | Low |
| 6 | No input validation in speak | Important | Low |
| 7 | No corpus size limit for LLM | Important | Medium |
| 8 | Non-atomic file copy / no sync | Important | Low |
| 9 | Manual TOML construction in clone | Important | Low |
| 10 | Backend re-registration issue | Important | Low |
| 11 | isStopWord per-call map allocation | Important | Trivial |
| 12 | Hardcoded 500ms delay | Important | Low |
| 13 | Non-deterministic Names() | Important | Trivial |
| 14 | No file type validation before upload | Important | Low |
| 15 | Low cmd/tts/watch test coverage | Important | High |
| 16 | Silent ExpandPath failure | Important | Low |
| 17 | Hardcoded "cyperx" identity | Nice-to-have | Medium |
| 18 | Silent style profile fallback | Nice-to-have | Trivial |
| 19 | Signal handler leak | Nice-to-have | Trivial |
| 20 | Swallowed read errors in corpus | Nice-to-have | Trivial |
| 21 | No CI / linter config | Nice-to-have | Low |
| 22 | No ElevenLabs retry/rate limit | Nice-to-have | Medium |
