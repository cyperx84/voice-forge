---
name: voice-forge
description: Build and use a user-owned communication style profile with the VoiceForge CLI. Use when importing voice transcripts or writing samples, analyzing how someone communicates, refreshing or exporting a portable voice profile, writing with confirmed style guidance, managing cloned voices, or generating speech audio.
---

# VoiceForge

Use VoiceForge as the owner of a person's private communication corpus and
derived style profile. Other tools may consume its portable export; they should
not read or copy the raw corpus unless the user explicitly requests it.

## Discover the installed interface

Run `forge --help` and the relevant subcommand's `--help` before acting. The CLI
is evolving, so prefer the installed interface over remembered flags.

## Core workflow

1. Check the environment with `forge doctor`.
2. Ingest only material the user has authorized:
   - `forge ingest <path>` for paired audio and transcripts.
   - `forge ingest-text <path>` for writing.
   - `forge ingest-code <path>` for coding-style signals.
   - `forge ingest-video <path>` or `forge ingest-photo <path>` when relevant.
   - `forge ingest-bulk ...` only after confirming every source directory.
3. Inspect corpus state with `forge stats` or `forge corpus --help`.
4. Run `forge analyze` to derive a style profile, or `forge refresh` to update an
   existing profile when the corpus has materially changed.
5. Review with `forge profile --brief` or `forge profile`.
6. Export the interoperability contract when another tool needs style guidance:
   `forge profile --output ./voice-profile.json`.

The portable file is versioned and conforms to
`schemas/voice-profile.schema.json`. Share that export instead of
`~/.forge/profile/style.json` or raw corpus entries.

## Using the profile

- Treat extracted patterns as evidence, not permanent identity.
- Preserve the user's meaning; never manufacture opinions to match a style.
- Confirm sensitive or surprising conclusions before relying on them.
- Use `forge write <topic>` only when the user wants style-assisted drafting.
- Use `forge speak <text>` only when the user requests audio generation.
- Inspect `forge backends` before selecting a TTS backend.

## Talk Once integration

Export a portable profile, then reference it from a Talk Once content package:

```json
{
  "voiceProfile": {
    "provider": "voiceforge",
    "path": "./private/voice-profile.json",
    "version": "1.0"
  }
}
```

VoiceForge owns the corpus and profile. Talk Once owns interviews, canonical
content, approvals, and derivatives.

## Safety

- Do not ingest broad directories without reviewing their contents and scope.
- Keep recordings, transcripts, derived profiles, and cloned voices private by
  default.
- Never publish, upload, or send generated text or audio without approval.
- Do not clone another person's voice without their explicit authorization.
- Report the profile path, export path, and commands run without exposing corpus
  contents in logs or summaries.
