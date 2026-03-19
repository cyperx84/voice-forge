# TTS Research Report 2026
**Voice Forge — TTS Technology State Assessment**  
*March 2026 | Target: Single user, M4 Mac Mini, ~60 min corpus of casual voice messages*

---

## Executive Summary

The TTS voice cloning landscape has shifted dramatically in the past 12 months. Open-source models have closed the gap with commercial APIs to the point where **Chatterbox Turbo** (MIT license, Dec 2025) is beating ElevenLabs in blind listening tests at 63.75% listener preference. For our use case — single user, M4 Mac Mini, ~60 min corpus of casual audio — the practical answer is a **tiered hybrid approach**: run Chatterbox or F5-TTS locally for cost-free generation, and keep ElevenLabs or Fish Audio as a cloud fallback when you need peak fidelity.

The biggest architectural insight: **zero-shot cloning from a short reference clip is now the dominant paradigm**. Fine-tuning a personal model on 60 minutes of audio is still valuable for long-form stability but is no longer required to get compelling results.

---

## 1. Local / Open-Source Models for Apple Silicon (M4)

### Tier 1 — Recommended

#### 🥇 Chatterbox Turbo (Resemble AI)
- **Model size:** 350M parameters
- **License:** MIT — fully commercial, no restrictions
- **Voice cloning:** Zero-shot from ≥5 seconds of reference audio
- **Latency:** Sub-200ms first audio (150ms typical), single-step mel decoder (10 diffusion steps → 1)
- **Quality:** 95/100 in benchmarks vs ElevenLabs Turbo's 90/100; **63.75% blind test preference over ElevenLabs** (Podonos, Dec 2025)
- **Apple Silicon:** Works via PyTorch MPS backend; not MLX-native but runs acceptably on M4
- **Extras:** Built-in PerTh watermarking (survives MP3 compression), supports STS (speech-to-speech)
- **Caveats:** Ecosystem still younger than XTTS; long-form narration consistency needs tuning; CUDA is the primary optimization target

```python
from chatterbox.tts_turbo import ChatterboxTurboTTS
tts = ChatterboxTurboTTS.from_pretrained()
audio = tts.generate(
    text="Your text here.",
    audio_prompt_path="reference_voice.wav"  # 5+ second clip
)
```

**Bottom line:** Best open-source voice cloning quality available as of March 2026. Start here.

---

#### 🥈 F5-TTS
- **Architecture:** Flow-matching based (ODE/diffusion hybrid), no reference transcript required for cloning
- **License:** Apache 2.0
- **Voice cloning:** Zero-shot from reference audio; cross-lingual cloning supported (arxiv:2509.14579, Sep 2025)
- **Quality:** "Close enough to ElevenLabs that the cost isn't worth it" (r/LocalLLaMA consensus, Aug 2025)
- **Apple Silicon:** MPS backend; runs well on M-series
- **Caveats:** Slightly behind Chatterbox on complex voice preservation per community benchmarks
- **Community:** Very active, high-quality ComfyUI and API server wrappers available

**Bottom line:** Solid #2 option. Preferred by many power users who ran XTTS for years and wanted an upgrade.

---

### Tier 2 — Situational Use

#### Kokoro (82M)
- **Architecture:** StyleTTS2-derived, 82M params
- **License:** Apache 2.0
- **Voice cloning:** ❌ No zero-shot voice cloning from audio samples — uses curated preset voice tensors only
- **Latency:** Generates 30 seconds of audio in under 1 second on GPU
- **Apple Silicon:** ✅ MLX-native via `mlx-audio` package — runs on Neural Engine/GPU, very fast
- **Quality:** High naturalness but "emotionless, AI delivery" in quality reviews
- **Best for:** Fast, low-resource TTS where you don't need to clone your specific voice

**Why it's relevant:** If Voice Forge needs a fast local TTS fallback that doesn't use your personal voice, Kokoro + mlx-audio is the most native M4 experience available. Real users have shipped full local voice pipelines with it (STT → LLM → Kokoro TTS entirely offline on M4).

```bash
pip install mlx-audio
python -m mlx_audio.tts.generate --model kokoro --text "Hello world" -o output.wav
```

---

#### Coqui XTTS v2
- **License:** Coqui Public Model License ⚠️ — restrictive for commercial products; free for non-commercial/research
- **Voice cloning:** Zero-shot from 6-second clip; accepts longer reference audio (better for complex voices)
- **Quality:** 75/100 in benchmarks — solid but outclassed by Chatterbox/F5-TTS now
- **Apple Silicon:** MPS supported; heavier than alternatives
- **Multilingual:** Excellent — one of the best for non-English cloning
- **Caveats:** License complexity is the main blocker for Voice Forge as a CLI tool; community more fragmented after Coqui's commercial pivot
- **When to use:** Best multilingual voice cloning if licensing is acceptable; community has built `xtts-api-server` for production-ready deployments

---

#### Fish Speech (local, open-source model)
- **License:** CC-BY-NC ⚠️ — non-commercial only
- **Model:** Fish Audio S1 (Oct 2025) — "most expressive and natural TTS model available" at launch
- **Voice cloning:** Zero-shot, instant
- **Apple Silicon:** MPS backend available
- **Caveats:** License blocks commercial use entirely; separate from Fish Audio cloud API (which has commercial terms)
- **When to use:** Personal/research use only; if licensing changes, this becomes a top-tier option

---

#### StyleTTS 2
- **License:** MIT
- **Voice cloning:** Supports zero-shot via style diffusion + WavLM speaker encoding
- **Quality:** Strong for narration/studio quality; Kokoro is built on this architecture
- **Caveats:** More complex setup than Chatterbox; less maintained since Kokoro superseded it for many use cases
- **Best for:** Custom fine-tuning pipelines where you want access to the style latent space directly

---

#### MetaVoice
- **Status:** Largely unmaintained as of 2026; development has stalled
- **Voice cloning:** 0-shot speaker adaptation
- **Recommendation:** Skip — use Chatterbox instead

---

#### Parler-TTS
- **Type:** Text-description controlled TTS — you describe the voice in natural language ("a warm male voice with slight reverb") rather than providing an audio reference
- **Voice cloning:** ❌ Not direct voice cloning from audio; different paradigm entirely
- **Use case:** Generating varied synthetic voices without a real reference; useful for data augmentation
- **Recommendation:** Not relevant for Voice Forge's cloning use case, but interesting for generating diverse training data

---

### Local Model Summary

| Model | Clone from Audio | License | M4 Performance | Quality Score | Recommendation |
|-------|:-:|---------|:-:|:-:|---|
| **Chatterbox Turbo** | ✅ 5s | MIT | Good (MPS) | 95/100 | **#1 — Use this** |
| **F5-TTS** | ✅ short clip | Apache 2.0 | Good (MPS) | ~88/100 | **#2 — Solid alt** |
| Kokoro | ❌ no cloning | Apache 2.0 | ✅ MLX-native | ~85/100 | Fast local TTS only |
| XTTS v2 | ✅ 6s+ | Coqui PML⚠️ | OK (MPS) | 75/100 | Non-commercial only |
| Fish Speech | ✅ instant | CC-BY-NC⚠️ | OK (MPS) | ~90/100 | Non-commercial only |
| StyleTTS 2 | ✅ | MIT | OK | ~80/100 | Complex; skip |
| MetaVoice | ✅ | Apache 2.0 | — | — | Unmaintained; skip |
| Parler-TTS | ❌ | Apache 2.0 | OK | — | Wrong paradigm |

---

## 2. Cloud TTS APIs for Voice Cloning

### 🥇 ElevenLabs — Gold Standard, Highest Quality
- **Instant Voice Clone:** 1-5 minutes of audio; ~85-90% accuracy
- **Professional Voice Clone:** 30+ minutes of studio audio; custom model trained on your voice
- **Pricing (March 2026):**
  - Free: 10K chars/month (~10 min audio)
  - Starter: ~$5/month, instant cloning
  - Creator: ~$22/month — **Professional Voice Cloning** + 100K chars/month
  - Pro: ~$99/month, 500K chars, 160 custom voices
- **Latency:** Flash v2.5 = 75ms first byte
- **Voice fidelity:** Best-in-class for cloned voices; captures prosody, timbre, emotional range
- **Caveats:** Highest cost; 30+ minutes requirement for professional clone means your casual corpus needs preprocessing to meet their quality bar

**For Voice Forge:** The Creator plan ($22/month) is the minimum tier to get Professional Voice Cloning. Your 60-minute corpus is perfect for this — exceeds the 30-minute minimum.

---

### 🥈 Fish Audio — Best Value
- **API pricing:** $15/million characters — **~80% cheaper than ElevenLabs**
- **Subscription:** Pro from $9.99/month (200 min generation)
- **Voice cloning:** Instant from short samples via S1 model
- **Quality:** Highly expressive S1 model (Oct 2025); community rates it highly for naturalness
- **Latency:** Competitive; sub-500ms typical
- **License:** Commercial use permitted via API

**For Voice Forge:** Best cost-per-character for high-volume use. If you're generating a lot of audio, Fish Audio's pricing is compelling. Quality is slightly below ElevenLabs professional clone but close.

---

### 🥉 Cartesia Sonic 3 — Best Latency for Real-Time
- **Latency:** 95ms first byte (Sonic); sub-200ms first chunk (Sonic 3, late 2025)
- **Pricing:** $46.70/million characters (expensive); free tier 10K credits, no commercial use
- **Voice cloning:** Instant from short audio samples; supports voice mixing and design
- **Best for:** Real-time conversational AI, gaming NPCs, voice agents where 2-way latency matters
- **Caveats:** Most expensive cloud option per character; voice clone fidelity is good but not ElevenLabs-tier for nuanced personal voices

---

### Resemble AI
- **Focus:** Enterprise-grade, ethical voice cloning with consent workflows
- **Pricing:** Custom/enterprise — less competitive for single-user
- **Notable:** They make Chatterbox (open-source), so their cloud product is essentially the same tech with SLA and compliance
- **For Voice Forge:** Skip unless you need enterprise features; use their open-source Chatterbox locally instead

---

### PlayHT
- **Status:** As of early 2026, PlayHT appears to have significantly reduced operations or shut down ⚠️. Multiple ElevenLabs blog posts reference "after PlayHT shutdown" in March 2026.
- **Recommendation:** Do not build a dependency on PlayHT

---

### Cloud API Summary

| Provider | Clone Quality | Latency | Pricing/1M chars | Best For |
|----------|:-:|:-:|:-:|---|
| **ElevenLabs** | ⭐⭐⭐⭐⭐ | 75ms | ~$330+ | Max fidelity personal clone |
| **Fish Audio** | ⭐⭐⭐⭐ | ~400ms | $15 | Best value; high volume |
| **Cartesia Sonic 3** | ⭐⭐⭐⭐ | 95ms | $46.70 | Real-time agents |
| Resemble AI | ⭐⭐⭐⭐ | ~400ms | Custom | Enterprise/compliance |
| ~~PlayHT~~ | — | — | — | ⚠️ May be defunct |

**Recommendation for Voice Forge:** Implement both ElevenLabs and Fish Audio providers. ElevenLabs as the high-fidelity option; Fish Audio as the cost-efficient default. Cartesia if you ever need real-time streaming.

---

## 3. Corpus Best Practices

### How Much Audio Do You Actually Need?

The "right answer" depends on the approach:

| Tier | Audio Required | Expected Quality | Approach |
|------|:-:|:-:|---|
| Zero-shot / Instant clone | 5-30 seconds | ~80% | Reference clip passed at inference time |
| Basic instant clone | 1-3 minutes | ~85-90% | ElevenLabs Instant / Fish Audio |
| Good personal clone | 10-30 minutes | ~92-95% | Fine-tuning or professional clone |
| **Excellent personal clone** | **30-60 minutes** | **~97%+** | ElevenLabs Professional / XTTS fine-tune |
| Diminishing returns | >60 minutes | marginal gain | Over-engineering |

**Your 60-minute corpus is exactly at the sweet spot for professional-grade cloning.** It exceeds ElevenLabs' Professional minimum (30 min) and is enough to fine-tune local models.

However: **quality of audio matters more than quantity past ~30 minutes.** 30 clean minutes outperforms 60 noisy minutes.

---

### Sample Rate & Format

- **Minimum:** 16kHz, 16-bit mono WAV — works for most models
- **Recommended:** 24kHz or 44.1kHz, 16-bit, mono WAV/FLAC
- **Avoid:** MP3 (lossy artifacts confuse speaker encoders), stereo (convert to mono), variable bitrate
- **Target SNR:** >30dB signal-to-noise ratio; ideally >40dB for fine-tuning datasets

```bash
# Normalize to 24kHz mono WAV using ffmpeg
ffmpeg -i input.m4a -ar 24000 -ac 1 -sample_fmt s16 output.wav
```

---

### Clean/Scripted vs. Natural Conversation

**The tension:** Your corpus is casual voice messages — naturalistic, potentially noisy, with filler words and informal prosody. This is actually a *feature* for voice cloning your natural speaking style, but it requires more preprocessing work.

**Guidance:**

| Use Case | Prefer |
|----------|--------|
| Cloning your natural speaking style | Natural conversation |
| Narration / long-form content | Scripted or semi-scripted |
| Real-time voice agents / conversational AI | Natural conversation |
| Audiobooks / formal TTS | Scripted + professional recording |

For Voice Forge's use case (personal voice clone, casual corpus): **natural conversation is fine and actually preferable** — it captures your real prosody, rhythm, and style. The preprocessing pipeline needs to do more work to extract the clean segments, but the output will sound more like *you* in natural contexts.

---

### Preprocessing Pipeline Recommendations

**Stage 1: Format Normalization**
```bash
# Batch normalize to 24kHz mono 16-bit WAV
for f in *.m4a *.mp4 *.mp3 *.ogg; do
  ffmpeg -i "$f" -ar 24000 -ac 1 -sample_fmt s16 "${f%.*}.wav"
done
```

**Stage 2: Noise Reduction**
- **Tool:** `demucs` (Meta's source separation) for music/background removal
- **Tool:** `noisereduce` Python library for stationary noise
- **Tool:** `resemblyzer` for speaker verification (filter out non-you audio)
- Filter rule: discard segments where SNR < -30dB in 1-second surrounding context

```python
import noisereduce as nr
import librosa

audio, sr = librosa.load("input.wav", sr=24000, mono=True)
reduced = nr.reduce_noise(y=audio, sr=sr, prop_decrease=0.8)
```

**Stage 3: Voice Activity Detection (VAD)**
- **Tool:** Silero VAD (lightweight, ONNX, runs on CPU) — best for Apple Silicon
- **Tool:** WebRTC VAD — fast but less accurate
- **Parameters:** Min speech segment 1.5s, max 15s, merge gaps <0.3s

```python
import torch
model, utils = torch.hub.load('snakers4/silero-vad', 'silero_vad')
(get_speech_timestamps, _, read_audio, *_) = utils
wav = read_audio('input.wav', sampling_rate=16000)
timestamps = get_speech_timestamps(wav, model, sampling_rate=16000)
```

**Stage 4: Segmentation & Quality Filtering**
- Target segment length: 3-15 seconds (most models prefer 5-10s)
- Discard: segments <1.5s, segments with SNR < 30dB, segments with background music
- Accept: natural pauses, um/uh (adds realism to cloned voice)

**Stage 5: Transcription (for models that need it)**
- **Tool:** Whisper large-v3 or faster-whisper (distil-large-v3 is fast on M4)
- Use for XTTS v2 (needs reference transcript), StyleTTS 2 fine-tuning
- F5-TTS and Chatterbox **do not** need transcripts for zero-shot reference

```bash
faster-whisper input.wav --model distil-large-v3 --output_format json
```

**Stage 6: Speaker Verification (optional but valuable)**
- If corpus includes other speakers (phone calls, etc.)
- **Tool:** `resemblyzer` — cosine similarity check against a clean reference sample of your voice
- Discard segments with similarity < 0.75

**Complete Pipeline Summary:**
```
raw audio → ffmpeg normalize → noise reduction → Silero VAD → 
segmentation (3-15s clips) → quality filter (SNR) → 
speaker verification → Whisper transcription → corpus/
```

---

### For Voice Forge CLI Integration

The preprocessing pipeline maps naturally to Voice Forge commands:

```
voice-forge corpus ingest --dir ./recordings --normalize --vad --denoise
voice-forge corpus segment --min 3s --max 15s --snr-threshold 30
voice-forge corpus verify --speaker-ref ./clean_reference.wav
voice-forge corpus stats  # show duration, segment count, SNR distribution
```

Key metrics to expose: total clean duration, segment count, average SNR, speaker similarity distribution.

---

## 4. Emerging Approaches

### Zero-Shot Voice Cloning — State of the Art

The paradigm has fundamentally shifted. In 2024, "voice cloning" meant fine-tuning a model on hours of audio. In 2026, SOTA is:

1. **Reference-clip inference** — pass 5-30 seconds at generation time, no training required
2. **Prompt-conditioned synthesis** — describe the voice in natural language (Qwen3-TTS, Parler-TTS)
3. **RL-enhanced quality** — reinforcement learning for naturalness (GLM-TTS, Dec 2025)

**Notable emerging models (March 2026):**

#### Qwen3-TTS (Alibaba, Jan 2026)
- Apache 2.0 license
- Natural language voice description: `"gruff pirate voice"` as input
- MLX-Audio port available for macOS: `pip install mlx-audio`
- Caveat: primarily CUDA-optimized; Mac MLX port is community-maintained and rough
- **Watch:** This model will matter more once the MLX port matures

#### SparkTTS (SparkAudio, 2025)
- BiCodec architecture: semantic tokens (content) + global tokens (speaker)
- Chain-of-thought generation with Qwen2.5 backbone
- SOTA zero-shot cloning with controllable prosody, pitch, speaking rate
- Still slower than Chatterbox on inference; quality improving rapidly

#### Sesame CSM (Conversational Speech Model)
- Designed for multi-turn conversational synthesis with context carry-over
- Maintains consistent voice across long sessions
- Relevant if Voice Forge moves toward conversational/interactive modes

#### IndexTTSv2
- Strong multilingual voice preservation; "best for complex voices" per r/LocalLLaMA
- Less latency-optimized than Chatterbox

#### Kyutai (Moshi)
- Streaming, real-time, bidirectional — designed for voice agents
- Impressive but has "drawbacks" noted by community (less stable cloning)

---

### Emotion Control

Current state (March 2026):
- **ElevenLabs:** Best emotional range via prompting and emotion tags in text
- **Chatterbox:** Good expressiveness, configurable "exaggeration" parameter controls emotional intensity
- **GLM-TTS:** RL-based multi-reward framework for emotional naturalness
- **SparkTTS:** CoT-based coarse/fine emotional control
- **ControlSpeech (arxiv:2406.01205):** Simultaneous zero-shot speaker cloning + zero-shot style/emotion control — research model but influential architecture

For Voice Forge: consider an `--emotion` flag that maps to model-specific controls (Chatterbox's exaggeration, ElevenLabs emotion tags).

---

### Style Transfer

- **DS-TTS (arxiv:2506.01020):** Dynamic dual-style feature modulation for zero-shot style adaptation from voice clips
- **StyleTTS 2:** Still the reference implementation for style latent manipulation
- **Key limitation:** Style transfer for extreme characteristics (dialects, specific emotional tones) is still unreliable across all models

**Architectural implication:** For Voice Forge's style extraction feature, the most robust approach is storing multiple reference clips organized by style/emotion rather than trying to extract a single style embedding. Use different clips as reference audio at inference time.

---

### What Changes the Architecture

**Critical insight for Voice Forge design:**

1. **Reference clips are the new model weights.** Instead of a trained personal model, maintain a library of high-quality reference clips (10-30 seconds each) organized by style/context. At generation time, select the appropriate reference for the target style.

2. **Zero-shot inference is fast enough to not pre-train.** The 150-200ms generation latency of Chatterbox Turbo means you can do reference-based generation in real-time without a pre-trained personal model.

3. **Hybrid local+cloud makes sense.** Local Chatterbox for draft/preview generation; cloud ElevenLabs Professional clone for final high-fidelity output.

4. **Transcript alignment enables better segmentation.** Using Qwen3-ForceAligner or WhisperX for word-level timestamps significantly improves corpus segmentation quality.

5. **Watermarking matters if you publish.** Chatterbox includes PerTh watermarking. If generating audio you'll share publicly, this is table stakes for ethical use.

---

## 5. Recommendations for Voice Forge (Ranked by Practicality)

### Immediate Actions (P0)

**1. Default local backend: Chatterbox Turbo**
- MIT license, best quality, zero-shot from reference clips
- Python subprocess wrapper from Go; return WAV/PCM audio
- Store 5-10 high-quality reference clips from corpus for different moods
- Config: `backend = "chatterbox-turbo"` with `reference_clip = "path/to/clip.wav"`

**2. Corpus preprocessing pipeline in Voice Forge**
- Add `voice-forge corpus preprocess` command
- Integrate: ffmpeg → noisereduce → Silero VAD → segmentation → quality filter
- Target: export clean 3-15s WAV clips + optional Whisper transcriptions
- SNR threshold: 30dB minimum for reference clips used in zero-shot

**3. ElevenLabs Professional Voice Clone provider**
- Creator plan ($22/month) unlocks Professional Voice Clone
- Upload preprocessed corpus (your 60 min → target ~30 min clean after filtering)
- API provider in Voice Forge for high-fidelity generation

### Near-Term (P1)

**4. Fish Audio provider**
- Implement as cost-efficient cloud fallback at $15/million chars
- Good quality, better pricing than ElevenLabs for bulk generation

**5. Reference clip library management**
- `voice-forge corpus style` — tag clips by context (excited, calm, explanatory, casual)
- At generation time, select reference based on target style
- This is your "style extraction" feature materialized

**6. mlx-audio / Kokoro integration for fast local TTS**
- Not for voice cloning but for fast draft-mode generation without GPU pressure
- `--mode fast` uses Kokoro; `--mode clone` uses Chatterbox with reference

### Watch List (P2)

**7. Qwen3-TTS MLX port**
- Once the macOS MLX port stabilizes (likely Q2 2026), it adds natural language voice description
- Could enable: `voice-forge generate --voice "warm, measured, slightly tired"` as style input

**8. SparkTTS for style controllability**
- BiCodec architecture may enable better style isolation than reference clips
- Worth evaluating once Python packaging stabilizes

**9. Cartesia Sonic 3**
- Only relevant if Voice Forge adds real-time streaming generation mode
- 95ms latency is unique; implement as optional streaming provider

---

## 6. Quick Reference Card

```
LOCAL CLONING (M4 Mac Mini)
├── Best quality:    Chatterbox Turbo (MIT, MPS, 5s ref, 150ms latency)
├── #2 alternative:  F5-TTS (Apache 2.0, flow-matching, MPS)
├── Fast local TTS:  Kokoro 82M (Apache 2.0, mlx-audio, no cloning)
└── Skip:            MetaVoice (unmaintained), Parler-TTS (wrong paradigm)

CLOUD APIS
├── Best fidelity:   ElevenLabs Professional ($22/mo, 30+ min corpus)
├── Best value:      Fish Audio ($15/M chars, good quality)
├── Lowest latency:  Cartesia Sonic 3 (95ms, real-time use cases)
└── Avoid:           PlayHT (may be defunct March 2026)

CORPUS (60 min casual voice messages)
├── Clean audio matters more than quantity past 30 min
├── Pipeline: ffmpeg → denoise → Silero VAD → 3-15s segments → SNR filter
├── Format: 24kHz mono 16-bit WAV
├── Target: ~30 min of clean segments from 60 min raw
└── Transcription: faster-whisper distil-large-v3 (fast on M4)

EMERGING TO WATCH
├── Qwen3-TTS: NL voice description, MLX port (Q2 2026)
├── SparkTTS: Style controllability via BiCodec
└── Sesame CSM: Conversational consistency across long sessions
```

---

## Sources & Further Reading

- Chatterbox Turbo release: https://www.resemble.ai/chatterbox/ (Dec 2025)
- Blind test results: https://byteiota.com/chatterbox-tts-open-source-voice-synthesis-beats-elevenlabs-2/
- F5-TTS cross-lingual: https://arxiv.org/abs/2509.14579 (Sep 2025)
- Voice Cloning Comprehensive Survey: https://arxiv.org/html/2505.00579v1 (May 2025)
- ElevenLabs vs Cartesia: https://elevenlabs.io/blog/elevenlabs-vs-cartesia (Mar 2026)
- F5-TTS vs Chatterbox community: https://www.reddit.com/r/StableDiffusion/comments/1lscvor/
- Best Audio Models Feb 2026: https://www.reddit.com/r/LocalLLaMA/comments/1r7bsfd/
- mlx-audio Apple Silicon: https://www.xugj520.cn/en/archives/mlx-audio-apple-silicon-tts-optimization.html
- Fish Audio S1: https://www.reddit.com/r/aicuriosity/comments/1obpzrn/
- Cartesia Sonic 3: https://getstream.io/blog/cartesia-sonic-3-tts/

---

*Report generated: March 19, 2026*  
*For Voice Forge — Go CLI for corpus management, style extraction, and pluggable TTS*
