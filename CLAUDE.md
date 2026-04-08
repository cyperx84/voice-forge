# CLAUDE.md

Guidance for Claude Code (and other AI assistants) working in this repository.

## What this project is

**Voice Forge** (`forge` binary) is a Go CLI for managing a personal voice/identity corpus, extracting speaking style via an LLM, and generating speech through multiple TTS backends. It also bridges to Gemini Live for real-time voice sessions.

- Module: `github.com/cyperx84/voice-forge`
- Go version: `1.25.1` (see `go.mod`)
- Binary entrypoint: `main.go` → `cmd.Execute()` (Cobra)
- Runtime home: `~/.forge/` (config, profile, voices, venvs, corpus.db)
- Config file: `~/.forge/config.toml` (auto-created with defaults on first run by `config.EnsureDefaults` via `PersistentPreRunE` in `cmd/root.go:16`)

## Build, test, run

```bash
# Build the binary (produces ./forge in repo root)
go build -o forge .

# Run all tests with coverage
go test ./...
go test ./... -cover

# Run a single package's tests
go test ./internal/corpus -run TestReadTranscripts

# Static checks
go vet ./...
```

There is **no Makefile, no CI workflow, and no linter config** (`.golangci.yml`) committed. When adding build/test automation, prefer adding a `Makefile` rather than scripts. A prebuilt `./forge` binary is checked in at the repo root — do NOT rebuild-and-commit it unless explicitly asked; `.gitignore` already excludes it for new work.

Install paths shipped to users:
- `go install github.com/cyperx84/voice-forge@latest`
- Homebrew formula at `homebrew-tap/Formula/voice-forge.rb` (the `cyperx84/tap` tap)

## Repository layout

```
voice-forge/
├── main.go                  # 15-line entrypoint
├── cmd/                     # Cobra command handlers (one file per command)
│   ├── root.go              # rootCmd + PersistentPreRunE (config bootstrap)
│   ├── speak.go             # TTS generation + preset transcoding
│   ├── analyze.go           # LLM style extraction (voice-only or multi-source)
│   ├── refresh.go           # Smart re-analysis with thresholds
│   ├── pipeline.go          # ingest → refresh → skill end-to-end + --watch
│   ├── watch.go             # long-lived ingest daemon
│   ├── ingest*.go           # ingest adapters (audio, text, code, photo, video, bulk)
│   ├── corpus.go            # multi-source corpus CLI (stats, search, recent, export)
│   ├── character.go         # character create/list/show
│   ├── live.go              # forge live (Gemini Live session manager)
│   ├── clone.go             # voice cloning via a TTS backend
│   ├── doctor.go            # environment + backend diagnostics
│   ├── backends.go          # `forge backends` listing
│   ├── skill.go             # OpenClaw skill file generator
│   ├── preprocess.go        # audio preprocessing (resample, segment)
│   ├── score.go             # quality scoring per recording
│   ├── embed.go             # speaker embeddings
│   ├── export.go            # export corpus as LJSpeech etc.
│   ├── profile.go / stats.go / voices.go / write.go / refresh.go
│   └── *_test.go            # `package cmd` tests; share testhelpers_test.go
├── internal/                # Library packages — NOT importable outside the module
│   ├── config/              # TOML config: Load, Save, ExpandPath, per-field helpers
│   ├── corpus/              # ReadTranscripts, ComputeStats, SQLite DB (`modernc.org/sqlite`)
│   ├── analyzer/            # LLM-based style analysis (single + multi-source)
│   ├── profile/             # Typed style profile load/save
│   ├── rewriter/            # Character-driven text rewriting via LLM
│   ├── character/           # Character definitions + builtin presets
│   ├── skill/               # Emits an OpenClaw skill package
│   ├── tts/                 # Backend interface + registry + 5 backends
│   ├── audioout/            # Output presets (discord, podcast, video, lossless) + listen page
│   ├── ffmpeg/              # Thin wrapper with thread / nice limits
│   ├── watch/               # fsnotify + poll-based ingest loop with sync.Mutex
│   ├── ingest/              # Per-type ingest adapters (text, code, photo, video, discord, twitter, social)
│   ├── live/                # Gemini Live bot process manager + character→prompt bridge
│   ├── discord/             # Discord voice file helpers
│   ├── preprocess/          # Audio normalization/segmentation
│   ├── scoring/             # Recording quality scoring
│   ├── embedding/           # Speaker embedding generation
│   └── export/              # Dataset exporters
├── scripts/setup-runtimes.sh  # Provisions isolated Chatterbox + F5 venvs
├── homebrew-tap/Formula/voice-forge.rb
└── README.md, SKILL.md, AUDIT.md, RESEARCH-*.md, ROADMAP-LIVE.md, TASK*.md
```

`cmd/` files each define their command as a package-level `var fooCmd = &cobra.Command{...}` and register it in their `init()` via `rootCmd.AddCommand(...)`. The exception is `cmd/live.go`, whose top-level `liveCmd` is wired from `cmd/root.go:27` explicitly.

## Key architectural patterns

### Command lifecycle
1. `main.go` calls `cmd.Execute()`.
2. Cobra dispatches to the selected command's `RunE`.
3. `PersistentPreRunE` on `rootCmd` calls `config.EnsureDefaults()`, which writes `~/.forge/config.toml` on first run.
4. Most commands then call `config.Load()` to get a `Config` struct with typed sub-configs.
5. Paths inside `Config` use `~/` prefixes; always expand via `config.ExpandPath(...)` or the typed accessors (`cfg.ProfileDir()`, `cfg.VoicesDir()`, `cfg.CorpusPaths()`, `cfg.CharactersDir()`, `cfg.SkillOutputDir()`, `cfg.CorpusDBPath()`, `cfg.WatchDir()`).

### TTS backend interface (`internal/tts/backend.go`)
All backends implement:
```go
Name() string
Speak(text string, opts SpeakOpts) ([]byte, error)
Clone(samples []string, name string) error
Available() bool
Setup() error
NativeFormat() AudioFormat
```
The registry is a global `map[string]Backend` guarded by `sync.RWMutex`. `Register()` is **idempotent** (no replacement if key exists — see `internal/tts/backend.go:43`). `ClearRegistry()` exists for tests. `Names()` returns a sorted slice.

Backends registered by `initBackends` in `cmd/speak.go:284`:
- `chatterbox` (local venv at `~/.forge/venvs/chatterbox`)
- `f5-tts` (local venv at `~/.forge/venvs/f5-tts`)
- `tts-toolkit` (external repo, path configurable)
- `kokoro` (dispatches through `tts-toolkit`)
- `elevenlabs` (cloud API, reads `ELEVENLABS_API_KEY` or config)

Python runtime resolution order is consistent across `speak`, `doctor`, and `backends`:
1. env var (`FORGE_CHATTERBOX_PYTHON` / `FORGE_F5_PYTHON`)
2. `[tts.chatterbox].runtime_path` / `[tts.f5].runtime_path` in config
3. default venv (`~/.forge/venvs/<backend>`)
4. fall back to system `python3`

See `tts.ResolveConfiguredRuntime(envVar, configPath, defaultSubpath)` in `internal/tts/python_runtime.go`.

### Audio output pipeline (`internal/audioout`)
Presets: `discord` (48kHz mono 128k mp3), `podcast` (44.1kHz stereo 192k mp3), `video` (48kHz stereo aac), `lossless` (44.1kHz mono 16-bit wav). When `--preset` or `--normalize` is set, `speak` writes the backend output to a temp WAV, then `audioout.Transcode` / `audioout.Normalize` produces the final file via ffmpeg using `cfg.FFmpeg.Threads` / `cfg.FFmpeg.Nice` limits. `--listen-link` additionally writes a self-contained HTML player next to the audio via `audioout.WriteListenPage`.

### Corpus: dual-mode (filesystem + SQLite)
- **Filesystem mode (V1+ legacy voice path):** transcripts are plain `.txt` files whose basename is a UUID (see `corpus.uuidPattern` at `internal/corpus/corpus.go:41`). `corpus.ReadTranscripts(paths)` walks every directory in `cfg.Corpus.Paths`, also reading a `transcripts/` subdirectory for the older `~/.openclaw/workspace/voice/` layout.
- **SQLite mode (V4+ multi-source):** `corpus.OpenDB(path)` opens `~/.forge/corpus.db` with WAL mode. Schema is a single `corpus_items` table (see `internal/corpus/db.go:53`) keyed by `id`, typed via the `Type*` constants in `internal/corpus/item.go` (`voice`, `text`, `video`, `photo`, `code`, `social`). `corpus.MigrateExistingCorpus(paths, db)` back-fills pre-DB voice files into the table.
- `cmd/analyze.go` prefers multi-source analysis when the DB exists, falling back to voice-only.

### Style profile lifecycle
1. `forge analyze` → `analyzer.Analyze(transcripts, llmCmd, llmArgs)` shells out to the `llm.command` (`claude --print --permission-mode bypassPermissions` by default) and parses a structured JSON response.
2. Output: `~/.forge/profile/style.json` (machine-readable) + `style-summary.md` (human-readable).
3. `forge refresh` checks age + growth thresholds (`Refresh.MinInterval`, `Refresh.MinNewTranscripts`, plus a 10% growth check); see `needsRefresh` at `cmd/refresh.go:80`. It guards division by zero when `SampleCount == 0`.
4. `forge speak --character <name>` calls `rewriter.Rewrite` which loads the style JSON via `rewriter.LoadStyleJSON` and composes a second LLM call to rewrite the input text in that character's voice — if the style profile fails to load, it warns and proceeds with `{}`.

### Watcher (`internal/watch/watch.go`)
- `Watcher` struct has a `sync.Mutex` guarding concurrent `ProcessExisting()` calls — do NOT remove this lock (see AUDIT.md finding #2 for the race it fixed).
- `Run(stop <-chan struct{})` combines fsnotify events with a poll ticker for NFS/remote-mount resilience.
- `FileWriteDelay` (default `500ms`) waits for the writer to finish after a file event; it is configurable via `[watch] file_write_delay`.
- Subprocess calls (ffmpeg, whisper) use `exec.CommandContext` with timeouts.

### Pipeline (`cmd/pipeline.go`)
- `runOncePipeline` — ingest → refresh → skill, all sequential.
- `runContinuousPipeline` (`--watch`) — runs `runOncePipeline` once, then starts `w.Run(stop)` plus a periodic goroutine that calls `runRefreshAndSkill` (NOT `ProcessExisting`) so the watcher owns ingest exclusively. Signal handlers are deregistered with `defer signal.Stop(sig)` on both `pipeline` and `watch`.

### Live (Gemini) integration
`forge live start|stop|status|voice|config` manage an external Gemini Live bot process via `internal/live`. `forge live start --character <name>` calls `live.CharacterToSystemPrompt(ch, stylePath)` to convert a loaded character + style profile into a system prompt before launching. Configuration lives under the `[live]` TOML section with nested `[live.discord]` and `[live.vad]` tables.

## Conventions and gotchas

### Error handling and logging
- Wrap errors with `fmt.Errorf("<what failed>: %w", err)`. Do not swallow `error` return values silently — if the call is a best-effort file scan (e.g. `corpus.ReadTranscripts`), `log.Printf("warning: ...")` before `continue`.
- User-facing progress output goes to `stdout`; warnings go to `stderr` (see `cmd/speak.go:82` for an example of `fmt.Fprintf(os.Stderr, "Warning: ...")`).
- `config.ExpandPath` logs a warning and returns the unexpanded path if `os.UserHomeDir()` fails — callers should treat this as best-effort.

### Path handling
- Always use `filepath.Join`, never string concatenation.
- Before writing, verify parent directories exist (`os.MkdirAll(dir, 0755)`) or use atomic temp-file-then-rename (see `cmd/ingest.go:98` `copyFile` for the canonical pattern).
- Config files written with `Save()` use mode `0600` (the ElevenLabs API key is stored there). Do not regress this to `0644`.

### Subprocess calls (claude, ffmpeg, whisper-cli)
- Use `exec.CommandContext(ctx, ...)` with a timeout. See `internal/watch/watch.go` for the pattern.
- The LLM command/args come from `cfg.LLM.Command` + `cfg.LLM.Args`; do not hardcode `"claude"` elsewhere.
- FFmpeg calls should go through `internal/ffmpeg` and honor `cfg.FFmpeg.Threads` and `cfg.FFmpeg.Nice`.

### Testing
- Test files live alongside their package (`foo_test.go`). `cmd/` tests use `package cmd` and share helpers in `cmd/testhelpers_test.go` (`testConfig()`).
- The `cmd` and `internal/tts` / `internal/watch` packages have historically low coverage (see `AUDIT.md` §15). When you touch code in these areas, prefer adding a test over none.
- Long-running or network-dependent tests should be gated behind short-circuiting env checks or `testing.Short()`. Do not assume `~/.forge` or `~/.openclaw` exist in tests — create temp dirs with `t.TempDir()`.
- Do not rely on `initBackends` being idempotent across tests: call `tts.ClearRegistry()` in `TestMain` or test setup when the test registers mock backends.

### Character identity and hardcoded user
- The default voice name, default character base, and preset list currently hardcode `"cyperx"` (see `AUDIT.md` §17). Treat new hardcodes of a specific identity as a code smell — prefer reading from `cfg.Voices.Default` / `cfg.Characters.Default`.

### Flag naming
- Boolean flags are positive (`--force`, `--watch`, `--discord`, `--listen-link`, `--voice-only`).
- `--discord` is an alias for `--preset discord` in `forge speak`.
- Long flags use kebab-case; short flags are scarce (only `-o` for `--output`).

### Commit style
Recent commit messages use conventional-ish prefixes: `feat:`, `fix:`, `docs:`. Short subject line, body explains "why". No Co-Authored-By trailers.

### Git workflow for this task
- Active development branch for AI changes: `claude/add-claude-documentation-ZF84o`.
- Never force-push to `main`. Create a new branch for any multi-commit change.
- Do not commit the `./forge` binary — add a line to `.gitignore` if it reappears in `git status`.
- Do not commit `~/.forge/config.toml` or any file under `~/.forge/`.

## Reference docs inside the repo

When planning a change that touches a given subsystem, read these first — they describe **intent**, not just current state:

- `README.md` — user-facing docs, install + config.
- `AUDIT.md` — production readiness audit (2026-03-19). Current open issues, severity, and fix suggestions. **Consult this before refactoring `cmd/`, `internal/tts`, `internal/watch`, `internal/config`.**
- `SKILL.md` — OpenClaw skill contract that `forge skill` emits.
- `ROADMAP-LIVE.md` — Voice Forge × Gemini Live integration phases (Phase 1 complete).
- `RESEARCH-ARCHITECTURE-2026.md` / `RESEARCH-TTS-2026.md` — background research snapshots.
- `TASK.md`, `TASK-V1..V5.md`, `TASK-PIPELINE.md`, `TASK-HARDEN.md` — historical task specs for each build milestone.

## Common workflows

```bash
# First-time setup
./scripts/setup-runtimes.sh   # provision Chatterbox + F5 venvs
forge doctor                  # verify environment
forge backends                # list which backends are actually available

# Daily use
forge ingest ./my-recording.ogg
forge analyze                 # or: forge refresh (cheaper)
forge stats
forge profile --brief

# TTS
forge speak "hello world" --voice cyperx --output ./out.wav
forge speak "ship it" --voice cyperx --preset discord --output ./out.mp3 --listen-link
forge clone --backend chatterbox --name cyperx

# Multi-source corpus
forge corpus stats
forge corpus search "query"
forge corpus recent --limit 20

# Long-running
forge watch                   # ingest daemon only
forge pipeline --watch        # ingest + refresh + skill, continuous

# Gemini Live
forge live config
forge live start --character cyperx
forge live status
forge live stop
```

## Things not to do

- Do not add a new TTS backend without also registering it in `cmd/speak.go:initBackends` and teaching `forge doctor` (`cmd/doctor.go:checkBackend`) how to report it.
- Do not bypass the config system by reading `~/.forge/...` directly — go through `config.Load()` and the typed accessors.
- Do not store the ElevenLabs API key at mode `0644`. `config.Save` writes `0600`.
- Do not call `exec.Command` without a timeout/context for external tools.
- Do not add new hardcoded `"cyperx"` / `"CyperX"` references; make them config-driven.
- Do not rebuild and commit the `./forge` binary.
- Do not create documentation files proactively — update this `CLAUDE.md` and the relevant `TASK-*.md` / audit doc instead.
