# Voice Forge — Pipeline Features (preprocess, score, export, embed)

## Context
Repo: `~/github/voice-forge`, Go + Cobra. V0-V3 already built. Read `RESEARCH-ARCHITECTURE-2026.md` and `RESEARCH-TTS-2026.md` for background. Build the missing pipeline stages.

## New Commands

### 1. `forge preprocess`
Normalize and clean corpus audio for optimal TTS training/cloning.

```
forge preprocess [--input ~/.forge/corpus/] [--output ~/.forge/processed/] [--force]
```

Pipeline per file:
1. **Format normalize**: Convert to 24kHz mono 16-bit WAV via ffmpeg
2. **Denoise**: Shell out to `noisereduce` Python tool or ffmpeg `afftdn` filter. Use ffmpeg's built-in filter to avoid Python dependency: `ffmpeg -i input.wav -af "afftdn=nf=-20" output.wav`
3. **VAD segmentation**: Use ffmpeg `silencedetect` to split on silence gaps > 0.5s, keep segments 3-15s. Write segments to `processed/segments/`
4. **Skip already-processed files** (check by modified time vs processed output)

Create `internal/preprocess/` package.

Output: processed directory with cleaned, segmented audio + a manifest JSON.

### 2. `forge score`
Quality-score each recording/segment.

```
forge score [--input ~/.forge/processed/] [--threshold silver]
```

Scoring criteria (compute via ffmpeg + basic audio analysis):
- **SNR** (signal-to-noise ratio): ffmpeg `astats` filter → extract RMS level + noise floor
- **Clipping %**: ffmpeg `astats` → `Peak count` / total samples
- **Duration**: 3-15s is gold, 1-3s or 15-30s is silver, outside is bronze
- **Silence ratio**: too much silence = lower score

Tier assignment:
- Gold: SNR > 30dB, no clipping, 3-15s, < 20% silence
- Silver: SNR > 20dB, < 0.1% clipping, 1-30s
- Bronze: SNR > 10dB, < 1% clipping
- Reject: below bronze thresholds

Output: scores JSON file with per-file tier + metrics. Print summary: "245 files scored: 89 Gold, 112 Silver, 38 Bronze, 6 Reject"

Create `internal/scoring/` package.

### 3. `forge export`
Export corpus in standard formats for TTS training.

```
forge export --format ljspeech [--tier gold,silver] [--output ~/.forge/export/]
```

Formats:
- **ljspeech**: `metadata.csv` (filename|transcript) + `wavs/` directory. Standard format most TTS models expect.
- **jsonl**: One JSON object per line with path, transcript, duration, tier, SNR

Only include files at or above the specified tier threshold.

Create `internal/export/` package.

### 4. `forge embed`
Generate voice embeddings for each recording using a speaker embedding model.

```
forge embed [--model ecapa-tdnn] [--reference <file>]
```

Implementation approach:
- Shell out to a Python one-liner using `speechbrain` or `resemblyzer`: 
  `python3 -c "from resemblyzer import VoiceEncoder; ..."` 
- If resemblyzer not installed, try speechbrain
- If neither available, print install instructions and exit gracefully
- Store embeddings in `~/.forge/embeddings/` as JSON (filename → 192-d float array)
- With `--reference`: compute cosine similarity of each recording to the reference
- Print summary: "Generated 245 embeddings. Mean self-similarity: 0.87"

Create `internal/embedding/` package.

## Config additions
```toml
[preprocess]
sample_rate = 24000
channels = 1
bit_depth = 16
denoise = true
min_segment = 3.0
max_segment = 15.0

[scoring]
default_threshold = "silver"

[export]
default_format = "ljspeech"
default_tier = "silver"

[embedding]
model = "resemblyzer"
```

## Tests
- Preprocess: test ffmpeg command construction, skip logic
- Score: test tier assignment logic with known values
- Export: test LJSpeech CSV format, tier filtering
- Embed: test embedding storage/retrieval, cosine similarity calc (mock the model)

## Git
- Use `--no-gpg-sign` for all commits
- ONE commit: "feat: pipeline stages — preprocess, score, export, embed"
- Push to `origin main` when done
- Run `go test ./...` and confirm all pass before committing

## When done
Run: `openclaw system event --text 'Done: Voice Forge pipeline — preprocess, score, export, embed built and pushed' --mode now`
