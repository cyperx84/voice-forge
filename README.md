# Voice Forge

Personal voice corpus management and style extraction CLI. Reads voice message transcripts, extracts your unique speaking style via LLM analysis, and makes it programmable.

## Install

```bash
go install github.com/cyperx84/voice-forge@latest
```

Or build from source:

```bash
git clone https://github.com/cyperx84/voice-forge.git
cd voice-forge
go build -o forge .
```

## Commands

### `forge analyze`

Reads all transcripts from your voice corpus, sends them to an LLM for style analysis, and outputs a structured profile.

```bash
forge analyze
```

**Output:**
- `~/.forge/profile/style.json` — machine-readable style profile (vocabulary, humor, argument style, emotional range, rhythm)
- `~/.forge/profile/style-summary.md` — human-readable markdown summary

### `forge stats`

Display corpus statistics: recording count, total/average duration, word counts, and top 50 words (excluding stop words).

```bash
forge stats
```

### `forge ingest <path>`

Import audio and transcript files into the voice corpus.

```bash
forge ingest ./my-recording.ogg
forge ingest ./recordings/          # import entire directory
```

Supported formats: `.ogg`, `.wav`, `.mp3`, `.m4a`, `.txt`

### `forge profile`

Pretty-print your current style profile.

```bash
forge profile           # full profile
forge profile --brief   # summary only
```

## Configuration

Config lives at `~/.forge/config.toml`. Created automatically on first run with sensible defaults.

```toml
[corpus]
paths = [
  "~/.openclaw/workspace/voice-corpus",
  "~/.openclaw/workspace/voice"
]

[llm]
command = "claude"
args = ["--print", "--permission-mode", "bypassPermissions"]

[profile]
output_dir = "~/.forge/profile"
```

## Testing

```bash
go test ./...
```

## License

MIT
