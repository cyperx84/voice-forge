# Voice Forge V2 — Character Mode

## Context
V0 (style extraction) and V1 (TTS backends + speak/clone/voices) are done. Repo: `~/github/voice-forge`, Go + Cobra. Style profile at `~/.forge/profile/style.json`. 3 TTS backends (tts-toolkit, elevenlabs, kokoro).

## What to Build

### 1. Character System (`internal/character/`)

Characters are style mutations on top of the base voice DNA. A character = base style profile + tone overrides + optional voice settings.

```go
type Character struct {
    Name        string            `json:"name" toml:"name"`
    Description string            `json:"description" toml:"description"`
    BasedOn     string            `json:"based_on" toml:"based_on"`     // "cyperx" or another voice
    ToneShift   ToneShift         `json:"tone_shift" toml:"tone_shift"`
    VoiceOpts   map[string]string `json:"voice_opts" toml:"voice_opts"` // backend-specific overrides
}

type ToneShift struct {
    Register    string   `json:"register"`     // "formal", "casual", "dramatic", etc.
    Pacing      string   `json:"pacing"`       // "slow", "measured", "fast", "breathless"
    Vocabulary  []string `json:"add_words"`    // words to add to the character's vocabulary
    AvoidWords  []string `json:"avoid_words"`  // words this character wouldn't use
    Persona     string   `json:"persona"`      // free-form persona description for LLM
    EmojiStyle  string   `json:"emoji_style"`  // "none", "minimal", "heavy"
}
```

### 2. Built-in Character Presets

Store in `~/.forge/characters/` as TOML files:

**narrator.toml** — Documentary narrator voice
- Register: measured, authoritative
- Pacing: slow, deliberate
- Drops slang, keeps technical terms
- Adds transitions: "What followed was...", "The result was clear..."

**podcast-host.toml** — Conversational podcast host
- Register: casual-professional
- Pacing: variable, energetic
- Keeps CyperX's enthusiasm but adds structure
- More "So here's the thing..." and "Let me break this down..."

**storyteller.toml** — Engaging storyteller
- Register: dramatic-casual
- Pacing: variable, builds tension
- Uses more sensory language
- Pauses for effect

**hype.toml** — Maximum energy CyperX
- Turns enthusiasm up to 11
- Short punchy sentences
- Heavy on action words
- "Let's go!", "Ship it!", energy

### 3. New Commands

**`forge character list`**
- List all characters (built-in + custom)
- Show name, description, base voice

**`forge character create <name>`**
- Interactive or flag-based character creation
- `--base cyperx --register formal --pacing slow --persona "Documentary narrator"`
- Saves to `~/.forge/characters/<name>.toml`

**`forge character show <name>`**
- Display character details

**`forge speak "text" --character narrator`**
- Apply character tone shift before TTS
- If the character has voice_opts, pass them to the TTS backend
- Text gets rewritten through the character's persona using LLM before speaking

**`forge write "topic" --character narrator`**  (NEW)
- Generate text in the character's voice (no audio)
- Uses style.json + character tone shift
- Outputs text to stdout
- `forge write "why I switched to Go" --character podcast-host`
- Uses LLM (claude --print) to generate text matching the character

### 4. Style-Aware Text Rewriting

`internal/rewriter/` package:
- Takes input text + style profile + character
- Shells out to `claude --print` with a prompt that includes the style DNA + character overrides
- Returns rewritten text in the character's voice
- Used by both `forge speak --character` and `forge write`

### 5. Config Updates
```toml
[characters]
dir = "~/.forge/characters"
default = ""  # optional default character
```

### 6. Tests
- Character loading/saving tests
- Rewriter mock tests (mock LLM output)
- Character list/show unit tests

### 7. Git
- Use `--no-gpg-sign` for all commits
- Push to `origin main` when done
- Small atomic commits

## Constraints
- Characters are TOML files — human-editable, simple
- LLM rewriting is optional (if no LLM configured, skip rewriting and use text as-is)
- Built-in presets are embedded in the binary but also written to ~/.forge/characters/ on first run
- Keep `forge write` simple — it's a text generator, not a full writing tool
- Pipe stdin to claude for LLM calls (same pattern as analyzer)
