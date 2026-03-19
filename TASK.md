# Build Task: Voice Forge V0

Read `/Users/cyperx/.openclaw/agents/builder/references/thread-voice-forge.md` for the full spec.

Build a Go CLI called `forge` with these commands:

1. `forge analyze` — reads .txt transcripts from `~/.openclaw/workspace/voice-corpus/` and `~/.openclaw/workspace/voice/transcripts/`, concatenates them, sends to LLM via `claude --print --permission-mode bypassPermissions` with a prompt to extract style profile. Outputs `~/.forge/profile/style.json` and `~/.forge/profile/style-summary.md`.

2. `forge stats` — corpus stats: recording count, total duration (parse manifest.txt format `<uuid>|<duration>|\n<transcript>`), word count, unique words, top 50 words excluding stop words.

3. `forge ingest <path>` — copy audio+transcript files into corpus dir.

4. `forge profile` — pretty-print style.json, with `--brief` flag.

Tech: Go, Cobra CLI, go-toml config at `~/.forge/config.toml`. Create repo with `gh repo create cyperx84/voice-forge --public --license mit --description "Personal voice corpus management and style extraction CLI"`. Module: `github.com/cyperx84/voice-forge`. Write tests. Good README. Push when done.

V0 only — no TTS, no voice cloning. Keep it simple.

When done, run: `openclaw system event --text "Voice Forge V0 built and pushed to cyperx84/voice-forge" --mode now`
