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

### `forge speak`

Generate speech from text with any configured TTS backend, including cloned voices saved under `~/.forge/voices`.

```bash
forge speak "quick draft" --voice cyperx --output ./out/draft.wav
forge speak "post this to Discord" --voice cyperx --discord --output ./out/discord-ready.mp3
forge speak "share this" --voice cyperx --discord --output ./out/share.mp3 --listen-link
```

Useful flags:
- `--discord` — normalize the final file to a conservative MP3 attachment (`48kHz`, mono, `128k`) via `ffmpeg` for more predictable Discord inline playback.
- `--listen-link` — generate `*.listen.html` beside the final audio file. The page embeds the audio directly, so a single HTML file can be hosted anywhere as a simple listen page.
- `--listen-title` — override the page title used by `--listen-link`.

Typical cloned-voice flow:

```bash
forge clone --backend chatterbox --name cyperx
forge speak "Status update: build is green." --voice cyperx --discord --output ./out/status.mp3 --listen-link
```

This gives you:
- `./out/status.mp3` — upload this directly to Discord
- `./out/status.listen.html` — host or open this as a dead-simple browser listen page

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

## Discord Notes

- `forge speak --discord` targets Discord's regular audio attachment player, not native Discord voice-message metadata.
- The generated `.mp3` is the safest default for playback as an uploaded file, but it will still appear as a normal attachment rather than a first-class voice note.
- `--listen-link` creates the HTML page locally. Voice Forge does not host it for you; you still need to upload that page somewhere if you want a public URL.
- Discord attachment and preview behavior still depends on client, channel permissions, and file-size limits.

## Testing

```bash
go test ./...
```

## License

MIT
