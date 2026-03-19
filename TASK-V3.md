# Voice Forge V3 — OpenClaw Integration

## Context
V0-V2 complete. Repo: `~/github/voice-forge`, Go + Cobra. Has: style extraction, 3 TTS backends, character mode with 4 presets, forge write/speak with character support. Style profile at `~/.forge/profile/style.json`. Binary at `~/.local/bin/forge`.

## What to Build

### 1. Auto-Capture: `forge watch`
A background watcher that monitors for new voice messages and auto-ingests them.

```
forge watch [--dir ~/.openclaw/workspace/voice-corpus/] [--interval 30s]
```

- Watches the voice-corpus directory for new `.ogg` files
- When a new file appears:
  1. Convert to WAV (ffmpeg)
  2. Transcribe (shell out to whisper-cli or existing whisper-transcribe script)
  3. Save transcript as `.txt` alongside the audio
  4. Add to corpus index
- Runs as a long-lived process (designed to be started by OpenClaw cron or systemd)
- Idempotent — skips files that already have transcripts
- Log output for monitoring

### 2. Auto-Analyze: `forge refresh`
Re-run style analysis on the full corpus to update the profile.

```
forge refresh [--force]
```

- Runs `forge analyze` but smarter:
  - Skip if profile is < 24h old and no new transcripts (unless --force)
  - Only re-analyze if corpus has grown by 10%+ or 20+ new transcripts
  - Update `~/.forge/profile/style.json` in place
  - Print diff summary: "Added 15 new transcripts, profile updated"
- Designed to be called by OpenClaw cron daily

### 3. OpenClaw Skill: `forge skill`
Generate/update an OpenClaw agent skill from the current style profile.

```
forge skill [--output ~/.openclaw/skills/cyperx-voice/]
```

- Reads `~/.forge/profile/style.json`
- Generates/updates the skill files:
  - `SKILL.md` — instructions for agents to write in CyperX's voice, using actual extracted data
  - `references/voice-profile.md` — the full style profile in readable format
  - `references/avoid-list.md` — words/phrases to never use
  - `references/key-phrases.md` — signature phrases and expressions
- The SKILL.md should instruct agents: "Read references/voice-profile.md to understand the voice. Use the key phrases naturally. Never use words from the avoid list."
- Overwrites existing skill files (the old hardcoded ones)

### 4. Pipeline Command: `forge pipeline`
Run the full pipeline end-to-end.

```
forge pipeline [--watch] [--skill-output ~/.openclaw/skills/cyperx-voice/]
```

Runs in order:
1. `forge ingest` — pick up any new files
2. `forge refresh` — re-analyze if needed
3. `forge skill` — update the OpenClaw skill
4. Print summary of what changed

With `--watch`: run continuously (watch + periodic refresh + skill update).

### 5. Config Updates
```toml
[watch]
dir = "~/.openclaw/workspace/voice-corpus/"
interval = "30s"
whisper_command = "whisper-cli"
whisper_model = ""  # auto-detect

[skill]
output = "~/.openclaw/skills/cyperx-voice/"
auto_update = true

[refresh]
min_interval = "24h"
min_new_transcripts = 20
```

### 6. Tests
- Watch: test file detection logic (mock filesystem)
- Refresh: test threshold logic
- Skill: test SKILL.md generation from a mock profile
- Pipeline: integration test

### 7. Git
- Use `--no-gpg-sign` for all commits
- Push to `origin main` when done
- Small atomic commits

## Constraints
- `forge watch` should be lightweight — no heavy polling, use fsnotify or similar
- Whisper transcription: shell out to whatever whisper binary is available (whisper-cli, whisper, mlx_whisper)
- The generated SKILL.md should be production-quality — this is what all agents will read
- Don't modify anything outside the voice-forge repo except writing to `~/.openclaw/skills/cyperx-voice/` when explicitly asked
- Keep the pipeline simple — it's glue, not a framework
