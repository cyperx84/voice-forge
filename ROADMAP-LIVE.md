# Voice Forge × Gemini Live — Integration Roadmap

Voice Forge is the studio. Gemini Live Discord is the stage.
One builds your voice identity, the other puts it live.

## Architecture

```
┌─────────────────────────────────────────────────────┐
│                    Voice Forge CLI                    │
│  forge analyze | speak | clone | ingest | live       │
├──────────┬──────────┬───────────┬───────────────────┤
│ Corpus   │ Profiles │ TTS       │ Live Engine       │
│ Manager  │ & Chars  │ Backends  │ (Gemini WS)       │
├──────────┴──────────┴───────────┴───────────────────┤
│              ~/.forge/ config + data                 │
└──────────────────┬──────────────────────────────────┘
                   │
        ┌──────────┴──────────┐
        │  Gemini Live API    │  (WebSocket, real-time)
        │  30 voices, 24 lang │
        └──────────┬──────────┘
                   │
        ┌──────────┴──────────┐
        │  Discord Voice      │  (or local mic, or HTTP)
        │  via discordgo      │
        └─────────────────────┘
```

## Phases

### Phase 1: Foundation (forge live + character bridge)
> The glue that connects Voice Forge identity to real-time voice.

- [ ] `forge live` command — manage Gemini Live sessions from Voice Forge
  - `forge live start` — start a live session (Discord or local mic)
  - `forge live stop` — stop the session
  - `forge live status` — show session state
  - `forge live config` — show/edit live config
- [ ] Character → system prompt bridge
  - Load `~/.forge/profile/style.json` or character file
  - Convert voice style traits → Gemini system prompt
  - `forge live start --character cyperx`
- [ ] Config integration in `~/.forge/config.toml`
  ```toml
  [live]
  gemini_api_key = ""  # or env: GEMINI_API_KEY
  model = "gemini-3.1-flash-live-preview"
  voice = "Orus"
  language = "en-US"

  [live.discord]
  token = ""  # or env: DISCORD_TOKEN
  voice_channel = ""
  guild = ""
  target_user = ""

  [live.vad]
  start_sensitivity = ""
  end_sensitivity = ""
  prefix_padding_ms = 20
  silence_duration_ms = 100
  ```
- [ ] Backward compat: `glive` CLI still works standalone

### Phase 2: Recording + Corpus Loop
> Every live session feeds back into your voice identity.

- [ ] Transcript capture during live sessions
  - Gemini returns text alongside audio — capture it
  - Save as timestamped transcript files
- [ ] Auto-ingest pipeline
  - `forge live start --record` — save session transcripts
  - On session end: `forge ingest` the transcript automatically
  - Corpus grows passively from conversations
- [ ] Session history
  - `forge live history` — list past sessions
  - `forge live replay <id>` — review what was said
- [ ] Metrics: words spoken, session duration, topics discussed

### Phase 3: Voice Agent Builder
> Define characters, deploy them as live voice agents.

- [ ] Character definitions for live agents
  - Extend `forge character` to include live-specific config
  - Voice selection, personality, behavioral constraints
  - `forge character create podcast-host --voice Puck --style upbeat`
- [ ] Multi-agent deployment
  - Spin up multiple bots with different characters
  - Each in a different Discord VC or same VC with user routing
  - `forge live start --character podcast-host --channel <id>`
- [ ] Character templates: customer support, podcast co-host, game NPC,
  study buddy, interview prep partner, language tutor

### Phase 4: Voice Style Transfer
> AI brain + your actual voice output.

- [ ] Hybrid pipeline: Gemini Live for understanding → forge speak for output
  - Gemini processes audio input, returns text intent
  - Text gets spoken through Chatterbox/F5 with your cloned voice
  - Latency tradeoff: slower but sounds like *you*
- [ ] `forge live start --voice-clone cyperx` 
  - Uses your cloned voice instead of Gemini's built-in voices
  - Falls back to Gemini voice if clone unavailable
- [ ] A/B mode: `forge live start --compare` 
  - Alternates between Gemini voice and cloned voice
  - Logs preference data for quality comparison

### Phase 5: Multi-Voice Scenes
> Multiple characters, multiple voices, one conversation.

- [ ] Scene definitions
  ```toml
  [scene.podcast]
  characters = ["host", "guest", "narrator"]
  
  [scene.podcast.host]
  voice = "Puck"
  style = "upbeat, asks questions"
  
  [scene.podcast.guest]
  voice = "Gacrux"  
  style = "knowledgeable, gives detailed answers"
  ```
- [ ] `forge live scene podcast --topic "AI voice cloning"`
  - Orchestrates multi-character conversation
  - Each character has distinct voice + personality
- [ ] Export scene audio as podcast episode
  - Multi-track or mixed-down single file
  - Chapter markers from turn boundaries

### Phase 6: Voice-First Command Interface
> Talk to control your tools.

- [ ] Intent detection layer
  - "Clone that last voice message" → `forge ingest`
  - "Read this back in my voice" → `forge speak`
  - "Switch to Puck" → voice swap
  - "How's my corpus looking?" → `forge stats`
- [ ] Plugin system for custom voice commands
- [ ] OpenClaw integration: voice commands that trigger agent actions

## Future Possibilities

- **Phone bridge** — Gemini Live session accessible via phone call (Twilio/SIP)
- **Meeting mode** — Join Google Meet/Zoom, act as AI participant
- **Streaming overlay** — Voice bot as stream co-host with OBS integration  
- **Voice journaling** — Talk freely, auto-transcribe + analyze + store in Obsidian
- **Language learning** — Tutor character that switches languages, corrects pronunciation
- **Accessibility** — Voice interface for people who can't type
- **API mode** — Expose live sessions over HTTP/WebSocket for web apps
- **Wake word** — "Hey Forge" activation instead of always-listening

## Technical Decisions

### Go native vs Node wrapper?
Phase 1: Wrap the existing Node bot via subprocess management.
Phase 2+: Migrate to pure Go using `discordgo` + native WebSocket.
Reason: Ship fast with what works, rewrite when the feature set stabilizes.

### Where does config live?
`~/.forge/config.toml` is the source of truth.
`.env` in the Node bot reads from it (or forge generates the .env).
`glive` CLI continues to work for quick edits.

### Multi-process or single binary?
Start multi-process (Go CLI manages Node bot).
Converge to single Go binary as features migrate.
