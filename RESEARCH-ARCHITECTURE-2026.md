# Voice Forge — Architecture Research Report 2026

> **Context:** Voice Forge is a Go CLI managing ~250 recordings (~60min audio), extracting writing style profiles via LLM, with pluggable TTS. This is gap-holding infrastructure for a larger personal voice system. This report covers what production voice systems look like and what Voice Forge should build next.

---

## Table of Contents

1. [Corpus Pipeline Architecture](#1-corpus-pipeline-architecture)
2. [Voice Fingerprinting & Embeddings](#2-voice-fingerprinting--embeddings)
3. [Style Transfer vs Voice Cloning](#3-style-transfer-vs-voice-cloning)
4. [Corpus Quality Scoring](#4-corpus-quality-scoring)
5. [Privacy & Data Sovereignty](#5-privacy--data-sovereignty)
6. [Integration Patterns](#6-integration-patterns)
7. [Recommendations & Roadmap](#7-recommendations--roadmap)

---

## 1. Corpus Pipeline Architecture

### How Production Systems Work

Production voice corpus pipelines follow a well-defined stage architecture. Voice Forge currently handles ingestion and style extraction but is missing the middle stages that make recordings actually usable for voice cloning.

```
Raw Audio
    │
    ▼
[INGESTION]
  ├── Format normalization (→ WAV 22050Hz 16-bit mono)
  ├── Metadata extraction (duration, channels, sample rate)
  └── Deduplication (by hash or near-duplicate detection)
    │
    ▼
[PREPROCESSING]
  ├── Denoising (RNNoise, DeepFilterNet, or speechbrain)
  ├── Dynamic range normalization (loudnorm to -23 LUFS)
  ├── DC offset removal
  └── Silence trimming (head/tail)
    │
    ▼
[SEGMENTATION]
  ├── VAD (Voice Activity Detection — silero-vad)
  ├── Long-form splitting (max 10–15 sec segments for TTS training)
  └── Silence gap detection
    │
    ▼
[ALIGNMENT]
  ├── STT transcription (Whisper large-v3)
  ├── Forced alignment (WhisperX → word-level timestamps)
  └── Transcript-audio pairing (LJSpeech format: metadata.csv + wavs/)
    │
    ▼
[QUALITY FILTERING]
  ├── SNR scoring (reject < 20dB)
  ├── Clipping detection (reject > 0.1% clipped samples)
  ├── Reverb scoring (SRMR)
  ├── Speaker verification (reject samples that don't match voice)
  └── MOS prediction (NISQA or speechmetrics)
    │
    ▼
[TRAINING-READY CORPUS]
  └── LJSpeech format, quality-ranked, tagged with metadata
```

### What Production Services Require

| Provider | Instant Clone | Professional Clone |
|---|---|---|
| ElevenLabs | 30 seconds | 30+ minutes |
| Resemble AI | 1 minute | 30–60 minutes |
| Cartesia | Few seconds | 15+ minutes |
| XTTS-v2 (local) | 6 seconds reference | Fine-tune on full corpus |
| Fish Speech (local) | Few seconds | Minimal reference needed |

**Voice Forge has ~60 minutes of audio** — this is in the "professional clone" tier for all major providers. That's a significant asset. The problem is the audio isn't preprocessed into a training-ready format.

### Open Source Training Targets (2025–2026)

The best local voice cloning options currently:

| Model | Clone Method | Quality | Corpus Format |
|---|---|---|---|
| **Coqui XTTS-v2** | Zero-shot (6s reference) + fine-tune | ⭐⭐⭐⭐ | LJSpeech (metadata.csv + wavs/) |
| **Fish Speech** | Zero-shot multilingual | ⭐⭐⭐⭐ | Audio + transcript pairs |
| **Chatterbox** (Resemble AI, MIT) | Zero-shot | ⭐⭐⭐⭐ | Reference audio clip |
| **Zonos** | Zero-shot | ⭐⭐⭐⭐ | Reference audio clip |
| **Orpheus 3B** | Zero-shot | ⭐⭐⭐ | Reference audio clip |
| **Higgs Audio V2** | Zero-shot | ⭐⭐⭐⭐⭐ | Reference audio clip |

**Recommendation:** Target LJSpeech format as the canonical output. It's supported by every serious TTS training framework. Voice Forge should have a `forge export --format ljspeech` command.

### What Voice Forge is Missing

| Stage | Missing? | Impact |
|---|---|---|
| Ingest + store | ✅ Have | — |
| Format normalization | ❌ Missing | Inconsistent sample rates crash training |
| Denoising | ❌ Missing | Background noise degrades clone quality |
| VAD-based segmentation | ❌ Missing | Long recordings aren't usable for training |
| Forced alignment | ❌ Missing | No word-level timestamps = can't build LJSpeech |
| Quality scoring | ❌ Missing | Bad recordings silently pollute the corpus |
| Export to training format | ❌ Missing | Can't hand off to any TTS training system |
| LLM style extraction | ✅ Have | — |

---

## 2. Voice Fingerprinting & Embeddings

### The Tools

**Speaker embeddings** compress a voice recording into a fixed-size vector that captures the speaker's acoustic identity, independent of what was said.

| Tool/Model | Embedding Size | Best For |
|---|---|---|
| **resemblyzer** | 256-d d-vector | Lightweight, fast, good baseline |
| **pyannote.audio** | Variable | Diarization, segmentation, full pipeline |
| **ECAPA-TDNN (SpeechBrain)** | 192-d | SOTA speaker verification |
| **WavLM-Large** | 768-d | Best overall, SSL pretrained |
| **TitaNet** | 192-d | NVIDIA NeMo ecosystem |

ECAPA-TDNN and WavLM are currently considered state-of-the-art for speaker verification tasks. ECAPA-TDNN is available via `speechbrain/spkrec-ecapa-voxceleb` on HuggingFace with 5 lines of Python.

### Why Store Embeddings Alongside Transcripts

**For a personal corpus, embeddings unlock four things:**

1. **Quality filtering**: Compute cosine similarity between each recording's embedding and your "gold" reference embedding. Recordings that drift far from your voice fingerprint (low similarity) are likely captured someone else speaking, phone calls where the other side leaked in, or heavily distorted recordings. Auto-reject these.

2. **Clone quality scoring**: Before/after comparison. Extract embeddings from reference recordings AND from TTS-generated audio. Cosine similarity between them gives you a measurable clone quality score. Track this over time as the corpus grows.

3. **Corpus drift detection**: Your voice changes over time (age, health, emotion). Embeddings can show you whether your current voice fingerprint has drifted from the training data. If drift is significant, older recordings may actively hurt clone quality.

4. **Speaker diarization on raw recordings**: If any recordings have multiple speakers (phone calls, videos with others), pyannote can segment by speaker. Only keep segments matching your embedding.

### Recommended Implementation

Store a `voice_embedding.npy` (192-d float32 ECAPA-TDNN vector) per recording in your corpus metadata. Add a `cosine_similarity_to_reference` score. Total storage overhead: ~800 bytes per recording × 250 = ~200KB. Negligible.

```json
// Extended corpus entry schema
{
  "id": "rec_20241201_142033",
  "path": "audio/rec_20241201_142033.wav",
  "transcript": "...",
  "duration_secs": 14.3,
  "embedding": "embeddings/rec_20241201_142033.npy",
  "speaker_similarity": 0.87,
  "quality_score": 0.91,
  "snr_db": 28.4,
  "created_at": "2024-12-01T14:20:33Z"
}
```

---

## 3. Style Transfer vs Voice Cloning

### The Critical Distinction

These are completely different problems operating on different modalities.

| Dimension | Writing Style Transfer | Voice Cloning |
|---|---|---|
| **Input** | Text transcripts | Audio waveforms |
| **Output** | Text that sounds like you wrote it | Audio that sounds like you spoke it |
| **Models** | LLM fine-tuning or prompting | TTS + acoustic model |
| **What it captures** | Vocabulary, rhythm, humor, sentence structure, phrases, ideas | Pitch, timbre, speaking rate, prosody, breath patterns |
| **What Voice Forge does** | ✅ Style extraction via LLM | Partially (pluggable TTS) |

Your transcripts are training data for the **writing** style layer. The actual audio is training data for the **acoustic** layer. Both are needed, but they're independent systems.

### The Unified Architecture

```
                    ┌─────────────────────────┐
                    │   Voice Forge Corpus    │
                    │  250 recordings, 60min  │
                    └────────┬────────────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
              ▼                             ▼
   [WRITING STYLE LAYER]         [ACOUSTIC VOICE LAYER]
   ├── Transcripts → LLM         ├── Audio → preprocessing
   ├── Style profile (JSON)      ├── Speaker embeddings
   ├── Vocabulary, rhythm        ├── Fine-tune TTS model
   └── Phrase patterns           └── Voice clone (XTTS-v2, etc.)
              │                             │
              └──────────────┬──────────────┘
                             │
                    ┌────────▼────────────┐
                    │   OUTPUT LAYER      │
                    │                    │
                    │  Input: topic/task  │
                    │       ↓             │
                    │  LLM generates text │
                    │  in YOUR writing    │
                    │  style              │
                    │       ↓             │
                    │  TTS renders audio  │
                    │  in YOUR voice      │
                    └────────────────────┘
```

### How the Two Systems Interact

The writing style and acoustic voice systems are **parallel pipelines that converge at synthesis time**. They don't need to directly communicate — the style profile informs the LLM prompt, and the voice clone model is loaded by the TTS engine. They meet only in the final step.

**The flow for generating content:**
1. User asks: "Write and narrate a summary of this article in my voice"
2. Style profile is injected into the LLM prompt: "Write this in the following style: [style_summary.md]"
3. LLM generates text that matches your writing patterns
4. Generated text is sent to TTS with your voice clone reference
5. Audio output sounds like you wrote AND spoke it

**Is there a unified approach?** Not yet at the model level, but architecturally, a single JSON profile document that combines both acoustic metadata and writing style is the clean solution. Voice Forge's `style.json` should grow to include acoustic fingerprint data alongside the existing vocabulary/rhythm/humor analysis.

---

## 4. Corpus Quality Scoring

### The Quality Metrics That Matter for TTS Training

| Metric | What It Measures | Target | Tool |
|---|---|---|---|
| **SNR** | Signal-to-noise ratio | > 20dB | `sox`, `pydub`, `librosa` |
| **Clipping** | Samples at ±1.0 (saturation) | < 0.1% | `librosa`, `soundfile` |
| **SRMR** | Speech-to-reverb ratio | > 3.0 | `speechmetrics` |
| **MOS-Net** | Predicted Mean Opinion Score | > 3.5 | `speechmetrics` |
| **NISQA** | Non-intrusive speech quality | > 3.0 | `NISQA` PyPI package |
| **Duration** | Segment length | 1–12 sec | `librosa.get_duration` |
| **Speaker similarity** | Match to reference embedding | > 0.75 | `resemblyzer`, ECAPA-TDNN |

### Python Scoring Snippet (Voice Forge can shell out to this)

```python
import librosa
import numpy as np
import soundfile as sf

def score_recording(path: str) -> dict:
    y, sr = librosa.load(path, sr=None, mono=True)
    
    # Clipping detection
    clip_pct = np.mean(np.abs(y) > 0.99) * 100
    
    # RMS-based SNR estimate
    rms = librosa.feature.rms(y=y)[0]
    noise_floor = np.percentile(rms, 10)
    signal_peak = np.percentile(rms, 90)
    snr_db = 20 * np.log10(signal_peak / (noise_floor + 1e-10))
    
    # Duration
    duration = librosa.get_duration(y=y, sr=sr)
    
    return {
        "snr_db": round(snr_db, 2),
        "clipping_pct": round(clip_pct, 4),
        "duration_secs": round(duration, 2),
        "sample_rate": sr,
        "is_usable": snr_db > 20 and clip_pct < 0.1 and 1 < duration < 30
    }
```

### For Full MOS Scoring

```bash
pip install speechmetrics NISQA
```

```python
import speechmetrics
metrics = speechmetrics.load('absolute', window=None)
scores = metrics(audio_path)
# scores['mosnet'] → predicted MOS (1-5 scale)
# scores['srmr'] → speech-to-reverb ratio
```

### Quality Tier Classification

Voice Forge should auto-classify recordings into tiers:

| Tier | Criteria | Usage |
|---|---|---|
| **Gold** | SNR > 30dB, no clip, MOS > 4.0, sim > 0.85 | Primary training data |
| **Silver** | SNR 20–30dB, clip < 0.05%, MOS > 3.5, sim > 0.75 | Supplementary training |
| **Bronze** | SNR 15–20dB | Reference only, not for fine-tuning |
| **Reject** | SNR < 15dB, clip > 0.1%, sim < 0.7 | Exclude from corpus |

**Prediction:** Based on the current ~250 recordings, 60% will likely be Gold/Silver tier, ~25% Bronze, ~15% Reject (assuming voice messages recorded on phone in varied environments). After quality filtering, you'll probably have ~35–40 minutes of usable training audio — well above the 30-minute threshold for professional clones.

### `forge score` Command Spec

```
forge score              # Score all recordings, update corpus metadata
forge score --min-tier silver  # Re-run with threshold filter
forge score --report     # Print quality distribution table
forge score --export-gold ./training/  # Export Gold tier in LJSpeech format
```

---

## 5. Privacy & Data Sovereignty

### The Legal Reality

Voice data is biometric data under multiple frameworks:

| Framework | Applies When | Voice = Biometric? | Key Requirements |
|---|---|---|---|
| **GDPR** (EU) | Data subject is EU resident | ✅ Yes | Explicit consent, right to deletion, encryption |
| **CCPA** (California) | Data subject is CA resident | ✅ Yes (special category) | Opt-in disclosure, deletion rights |
| **Illinois BIPA** | Subject in IL | ✅ Yes | Written consent, 3-year retention limit |
| **Privacy Act 1988** (AU) | Australian residents | Sensitive information | Consent required |

**For a personal corpus of your own voice:** The consent story is simple — you're both the data subject and the controller. You have implicit consent. The risk is low.

**The risk escalates if:**
- You ever use this system to clone someone else's voice (even with their permission — document it)
- The corpus is stored in a cloud service
- The system is exposed to others via API
- You use cloned voice for commercial purposes without disclosure

### Practical Sovereignty Measures

**1. Encrypt the corpus at rest**
```bash
# Full-disk encryption covers this on macOS (FileVault) — already done.
# For corpus portability, add per-directory encryption:
# Option A: age encryption for corpus archives
age -r <recipient_key> -o corpus.tar.age corpus.tar

# Option B: Store corpus in encrypted sparse image (macOS)
hdiutil create -size 2g -encryption AES-256 -fs HFS+ ./voice-corpus.dmg
```

**2. Never sync voice audio to unencrypted cloud storage**
- iCloud (with Advanced Data Protection): acceptable
- Dropbox standard tier: not recommended for biometric data
- Plain S3: not recommended without SSE-C

**3. Consent record template** (for anyone who's voice you record with permission)
```yaml
# consent_record.yaml
subject_name: "Chris Cyper"
is_self: true
recorded_by: "Chris Cyper"
purpose: "Personal AI voice assistant cloning"
date: "2024-01-01"
retention_years: 5
deletion_contact: "self"
```

**4. Data minimization**
- Store transcripts, not raw audio, wherever possible for style analysis
- Audio only needed for acoustic modeling — don't keep redundant copies
- Maintain a corpus inventory with hashes so you can prove data lineage

### Voice Watermarking (Future-proofing)
Tools like **Resemble Detect** (open source) can embed imperceptible watermarks in generated audio. If your voice clone is ever used without your consent, you can prove it's synthetic. Worth adding to the TTS output pipeline.

---

## 6. Integration Patterns

### How AI Agents Handle Voice

Three architectural patterns exist for voice-enabled AI agents:

```
CASCADE (most flexible, best for tools/function calling):
  User speaks → STT → LLM (text) → TTS → Audio out
  Providers: Whisper + Claude/GPT + ElevenLabs/Voice clone
  Latency: ~1.5-3s for local stack
  Voice clone: ✅ Easy (control TTS layer)

HALF-CASCADE (faster, preserves paralinguistic cues):
  User speaks → Multimodal LLM → TTS → Audio out
  Providers: Ultravox + ElevenLabs
  Latency: ~0.8-1.5s
  Voice clone: ✅ Possible at TTS layer

SPEECH-TO-SPEECH (fastest, most natural):
  User speaks → Multimodal LLM (audio in, audio out)
  Providers: OpenAI Realtime API, Google Live API
  Latency: ~400ms
  Voice clone: ⚠️ Limited support, closed ecosystem
```

**For "writes like me AND speaks like me"**, cascade is the only realistic architecture today. Speech-to-speech models don't support voice cloning well yet.

### The "Clone Me" Agent Pattern

```
                     ┌──────────────────┐
         Trigger     │   Trigger Source  │
         (text/voice)│  Discord, voice,  │
                     │  API, etc.        │
                     └────────┬─────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │  Style Profile   │  ← ~/.forge/profile/style.json
                    │  Injection       │
                    └────────┬─────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │  LLM Generation  │  ← Claude/GPT with style prompt
                    │  (text output)   │
                    └────────┬─────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │  Voice Synthesis  │  ← XTTS-v2 / Fish Speech / ElevenLabs
                    │  (cloned voice)  │  ← Reference: best Gold-tier recording
                    └────────┬─────────┘
                              │
                              ▼
                         Audio output
```

### Voice Agent Pipeline Libraries (2025)

For building the real-time conversation layer:

| Tool | Role | Notes |
|---|---|---|
| **LiveKit** | Real-time audio transport | Agents SDK, WebRTC |
| **Pipecat** | Pipeline orchestration | Excellent cascade support |
| **silero-vad** | Voice activity detection | CPU-friendly, <1ms |
| **WhisperX** | STT with forced alignment | Best for corpus building |
| **pyannote** | Speaker diarization | For filtering multi-speaker recordings |

### OpenClaw Integration

For the "Builder" agent context, the most direct integration pattern:

```go
// Future: forge speak "text to narrate"
// 1. Load style profile
// 2. Send text through LLM with style prompt (if rewriting needed)
// 3. Shell out to XTTS-v2 or Fish Speech with best Gold reference recording
// 4. Stream or save audio
```

---

## 7. Recommendations & Roadmap

### Immediate (v0.2 — Corpus Quality)

These have the highest ROI and don't require ML infrastructure.

**1. `forge score` — Automatic quality scoring**
- Shell out to Python script that scores each recording: SNR, clipping, duration
- Store scores in corpus metadata JSON
- Report quality distribution: Gold/Silver/Bronze/Reject
- Flag which recordings to exclude from TTS training

**2. `forge preprocess` — Audio normalization**
- Convert all recordings to WAV 22050Hz 16-bit mono (via `ffmpeg`)
- Apply loudnorm to -23 LUFS
- Trim leading/trailing silence (100ms threshold)
- This alone will improve TTS quality significantly

**3. `forge export --format ljspeech` — Training-ready output**
- Produce `metadata.csv` + `wavs/` directory
- Only export Gold + Silver tier by default
- Include estimated training time per TTS model

### Short-term (v0.3 — Voice Fingerprinting)

**4. `forge embed` — Speaker embeddings**
- Shell out to Python with ECAPA-TDNN (SpeechBrain)
- Store `.npy` embedding per recording
- Compute cosine similarity to reference embedding
- Add `speaker_similarity` field to corpus metadata
- Flag recordings with similarity < 0.75 for review

**5. `forge reference set <recording-id>` — Set canonical voice reference**
- Select the single best, cleanest recording as the reference voice
- All other recordings scored relative to it
- This reference also used as the TTS voice clone reference clip

### Medium-term (v0.4 — Style + Acoustic Unified)

**6. Extended style profile schema**
Merge acoustic metadata into the style profile:
```json
{
  "writing": { ... existing ... },
  "acoustic": {
    "reference_recording": "rec_20241201_142033",
    "reference_embedding": "embeddings/ref.npy",
    "corpus_quality_summary": {
      "gold_count": 140,
      "silver_count": 72,
      "bronze_count": 28,
      "reject_count": 10,
      "total_gold_silver_minutes": 38.4
    },
    "average_speaking_rate_wpm": 142,
    "pitch_range_hz": [95, 180]
  }
}
```

**7. `forge speak "text"` — End-to-end narration**
- Text → (optional style rewrite via LLM) → XTTS-v2 with Gold reference → audio
- This closes the loop from corpus management to actual usage

### Long-term (v1.0 — Production Voice System)

**8. TTS fine-tuning pipeline**
- Export Gold-tier corpus in LJSpeech format
- One-command fine-tuning: `forge train --model xtts-v2 --epochs 100`
- Evaluate clone quality with speaker similarity scoring

**9. Real-time voice agent mode**
- `forge agent --voice` — starts a local cascade voice agent
- Listens via silero-vad, transcribes with Whisper, generates with style-prompted LLM, speaks in your voice clone
- Integration with OpenClaw as a voice interface

**10. Privacy hardening**
- `forge encrypt` — AES-256 encrypt the corpus directory
- `forge audit` — report what data is stored, where, and flag any cloud sync risks
- Optional: add Resemble Detect watermarking to all generated audio

---

## Summary: Current State vs Target State

| Capability | Now | Target (v0.4) |
|---|---|---|
| Ingest audio + transcripts | ✅ | ✅ |
| Style profile (LLM) | ✅ | ✅ Enhanced |
| Audio quality scoring | ❌ | ✅ `forge score` |
| Audio preprocessing | ❌ | ✅ `forge preprocess` |
| Speaker embeddings | ❌ | ✅ `forge embed` |
| Export to LJSpeech | ❌ | ✅ `forge export` |
| Voice reference management | ❌ | ✅ `forge reference` |
| End-to-end narration | ❌ | ✅ `forge speak` |
| Corpus encryption | ❌ | ✅ `forge encrypt` |
| TTS fine-tuning | ❌ | 🔄 v1.0 |
| Real-time voice agent | ❌ | 🔄 v1.0 |

---

## Key Technical Choices

| Decision | Recommendation | Rationale |
|---|---|---|
| TTS model for local use | **XTTS-v2** (fine-tune) + **Chatterbox** (zero-shot) | XTTS-v2 is best for fine-tuning; Chatterbox (MIT license) is best for instant cloning |
| Speaker embeddings | **ECAPA-TDNN** via SpeechBrain | SOTA, HuggingFace available, 192-d compact vectors |
| STT + alignment | **WhisperX** | Word-level timestamps + diarization in one tool |
| Audio quality | **speechmetrics** + **librosa** | MOSNet + SRMR + SNR covers all quality dimensions |
| Corpus format | **LJSpeech** | Universal compatibility with all training frameworks |
| Privacy | **macOS FileVault** (already on) + **no unencrypted cloud sync** | Pragmatic baseline for personal corpus |

---

*Generated: March 2026 | Voice Forge Research | For internal use*
