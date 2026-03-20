# TASK: Voice Forge V4 — Multi-Source Identity Corpus

## Context

Voice Forge currently ingests voice messages (.ogg) and transcripts (.txt) only.
V4 expands the corpus to accept ANY media that represents a person's identity:
text, video, photos, code, social media posts — all owned locally.

The goal: build a universal identity corpus that captures how someone writes,
speaks, codes, presents visually, and thinks. Then use that corpus to create
characters that can generate content in any of those modalities.

## Architecture

### Current corpus structure (V0-V3)
```
~/.forge/corpus/
  voice-corpus/
    uuid.ogg    # voice message
    uuid.wav    # converted audio
    uuid.txt    # transcript
```

### V4 corpus structure
```
~/.forge/corpus/
  voice/           # existing voice messages (ogg/wav/txt triplets)
  text/            # written content
    *.md            # blog posts, notes, Discord messages, tweets
  video/           # video content
    *.mp4           # video files
    *.vtt           # subtitles/transcripts
    *.json          # metadata (duration, source, description)
  photos/          # visual identity
    *.jpg|png|webp  # photos
    *.json          # metadata (tags, context, source)
  code/            # coding style
    *.{ts,go,py,rs} # code samples
    *.json          # metadata (language, repo, context)
  social/          # social media posts
    *.json          # structured post data (text, media refs, engagement)
```

### Universal Corpus Item

Every item in the corpus gets a common metadata envelope:

```json
{
  "id": "uuid",
  "type": "voice|text|video|photo|code|social",
  "source": "discord|twitter|blog|github|local|...",
  "created_at": "ISO-8601",
  "ingested_at": "ISO-8601",
  "path": "relative path to primary file",
  "transcript": "text content or transcript if applicable",
  "tags": ["tag1", "tag2"],
  "metadata": {}  // type-specific metadata
}
```

Store in SQLite: `~/.forge/corpus.db`

## New Commands

### `forge ingest-text`
Ingest written content into the corpus.

```bash
# Ingest a single file
forge ingest-text ~/blog/my-post.md --source blog

# Ingest Discord messages (export JSON)
forge ingest-text ~/discord-export.json --source discord --format discord-export

# Ingest from stdin
echo "some text" | forge ingest-text --source note

# Ingest tweets (from xurl export or JSON)
forge ingest-text ~/tweets.json --source twitter --format twitter-archive
```

### `forge ingest-video`
Ingest video content. Extracts: transcript (whisper), keyframes, metadata.

```bash
# Ingest a video file
forge ingest-video ~/Videos/talk.mp4 --source local

# Ingest YouTube video (download + process)
forge ingest-video "https://youtube.com/watch?v=..." --source youtube

# Just extract transcript, skip keyframes
forge ingest-video ~/clip.mp4 --transcript-only
```

Implementation:
- Use ffmpeg for frame extraction (1 frame per 10s or scene-change detection)
- Use whisper for transcript
- Store video metadata (duration, resolution, codec)

### `forge ingest-photo`
Ingest photos with optional tagging.

```bash
# Ingest a photo
forge ingest-photo ~/Photos/headshot.jpg --tags "profile,headshot"

# Ingest a directory
forge ingest-photo ~/Photos/brand/ --source brand-kit --recursive

# With auto-tagging (uses LLM vision)
forge ingest-photo ~/photo.jpg --auto-tag
```

### `forge ingest-code`
Ingest code samples to capture coding style.

```bash
# Ingest files from a repo
forge ingest-code ~/github/voice-forge/ --language go

# Ingest specific files
forge ingest-code main.go utils.go --source voice-forge

# Ingest from GitHub directly
forge ingest-code --github cyperx84/voice-forge --language go
```

### `forge corpus`
Unified corpus management (replaces/extends `forge stats`).

```bash
# Full corpus overview
forge corpus stats

# Stats by type
forge corpus stats --type voice
forge corpus stats --type text

# Search across all corpus types
forge corpus search "snowboard"

# List recent additions
forge corpus recent --limit 20

# Export corpus manifest
forge corpus export --format json
```

### `forge analyze` (extended)
Update `forge analyze` to process ALL corpus types, not just voice transcripts.

The style profile should now include:
- **voice_style**: existing voice DNA (from transcripts)
- **writing_style**: patterns from text corpus (sentence structure, vocabulary, formatting preferences)
- **coding_style**: patterns from code corpus (naming conventions, comment style, architecture preferences)
- **visual_style**: tags/themes from photo corpus (color preferences, composition, aesthetic)
- **content_themes**: recurring topics across all sources

## Implementation

### New packages

```
internal/
  corpus/
    db.go          # SQLite corpus database
    item.go        # Universal CorpusItem type
    migrate.go     # DB migrations
  ingest/
    text.go        # Text ingestion (markdown, JSON exports)
    video.go       # Video ingestion (ffmpeg + whisper)
    photo.go       # Photo ingestion (copy + metadata)
    code.go        # Code ingestion (language detection, sampling)
    social.go      # Social media format parsers
    discord.go     # Discord export parser
    twitter.go     # Twitter archive parser
  analyzer/
    multi.go       # Multi-source analysis (extends existing analyzer)
    writing.go     # Writing style analysis
    coding.go      # Code style analysis
```

### SQLite schema

```sql
CREATE TABLE corpus_items (
  id TEXT PRIMARY KEY,
  type TEXT NOT NULL,          -- voice, text, video, photo, code, social
  source TEXT NOT NULL,        -- discord, twitter, blog, github, local, etc.
  created_at TEXT,
  ingested_at TEXT NOT NULL,
  path TEXT NOT NULL,          -- relative to corpus root
  transcript TEXT,             -- text content or transcript
  tags TEXT,                   -- JSON array
  metadata TEXT,               -- JSON object, type-specific
  word_count INTEGER DEFAULT 0,
  duration_seconds REAL,       -- for audio/video
  file_size INTEGER
);

CREATE INDEX idx_type ON corpus_items(type);
CREATE INDEX idx_source ON corpus_items(source);
CREATE INDEX idx_created ON corpus_items(created_at);
CREATE INDEX idx_tags ON corpus_items(tags);  -- FTS later
```

### Migration from V0-V3

The existing voice corpus files should be auto-migrated into the SQLite DB
on first run. Don't move files — just index them in the DB with their
existing paths.

```go
func MigrateExistingCorpus(cfg *config.Config, db *DB) error {
    // Scan voice-corpus dirs for .ogg/.wav/.txt triplets
    // Create corpus_items entries with type="voice"
    // Preserve existing file locations
}
```

## Config additions

```toml
[corpus]
root = "~/.forge/corpus"       # corpus root directory
db = "~/.forge/corpus.db"      # SQLite database path

[ingest]
auto_tag = false               # use LLM for auto-tagging photos
whisper_command = "whisper-cli" # STT command
video_keyframe_interval = 10   # seconds between keyframe extractions

[ingest.sources]
discord_export = ""            # path to Discord data export
twitter_archive = ""           # path to Twitter archive
```

## Testing

- Unit tests for each ingest adapter (text, video, photo, code, social)
- Unit tests for SQLite corpus DB operations
- Unit tests for Discord/Twitter export parsers
- Integration test: ingest mixed content → analyze → verify multi-source profile
- Migration test: existing voice corpus → SQLite indexing

## Constraints

- All data stays local. No cloud uploads unless explicitly configured.
- SQLite for the DB — simple, portable, no server.
- Don't break V0-V3 commands. Existing `forge stats`, `forge analyze`, etc. should continue working.
- The multi-source profile extends the existing style.json, doesn't replace it.
- Keep ingest adapters pluggable — easy to add new sources later.
- Use ffmpeg for all audio/video processing (already a dependency).
- Use the configured whisper command for any new transcription needs.
- Don't auto-download anything without explicit user action (no scraping).

## Success criteria

1. `forge ingest-text some-file.md` works and indexes in SQLite
2. `forge ingest-code ~/github/repo/` scans and indexes code samples
3. `forge corpus stats` shows breakdown by type
4. `forge corpus search "keyword"` searches across all types
5. `forge analyze` produces an extended multi-source profile
6. Existing voice corpus auto-migrates on first run
7. All existing V0-V3 commands still work unchanged
8. Tests pass for all new packages
