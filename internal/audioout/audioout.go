package audioout

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/cyperx84/voice-forge/internal/ffmpeg"
)

// Preset defines an output format profile for audio transcoding.
type Preset struct {
	Name       string
	SampleRate int
	Channels   int
	Codec      string
	Bitrate    string // empty for lossless codecs
	Format     string // file extension without dot
}

// Presets maps preset names to their configurations.
var Presets = map[string]Preset{
	"discord":  {Name: "discord", SampleRate: 48000, Channels: 1, Codec: "libmp3lame", Bitrate: "128k", Format: "mp3"},
	"podcast":  {Name: "podcast", SampleRate: 44100, Channels: 2, Codec: "libmp3lame", Bitrate: "192k", Format: "mp3"},
	"video":    {Name: "video", SampleRate: 48000, Channels: 2, Codec: "aac", Bitrate: "192k", Format: "m4a"},
	"lossless": {Name: "lossless", SampleRate: 44100, Channels: 1, Codec: "pcm_s16le", Format: "wav"},
}

// PresetNames returns sorted preset names for display.
func PresetNames() []string {
	return []string{"discord", "lossless", "podcast", "video"}
}

// FFmpegArgs returns the ffmpeg arguments for transcoding with this preset.
func (p Preset) FFmpegArgs(inputPath, outputPath string) []string {
	args := []string{
		"-y",
		"-i", inputPath,
		"-vn",
		"-ar", strconv.Itoa(p.SampleRate),
		"-ac", strconv.Itoa(p.Channels),
		"-codec:a", p.Codec,
	}
	if p.Bitrate != "" {
		args = append(args, "-b:a", p.Bitrate)
	}
	args = append(args, outputPath)
	return args
}

// Transcode converts audio using a preset and ffmpeg config.
func Transcode(inputPath, outputPath string, preset Preset, ffCfg ffmpeg.Config) error {
	args := preset.FFmpegArgs(inputPath, outputPath)
	return ffmpeg.RunSilent(ffCfg, args...)
}

// TranscodeDiscordMP3 converts audio into a Discord-friendly MP3 attachment.
func TranscodeDiscordMP3(inputPath, outputPath string, ffCfg ffmpeg.Config) error {
	return Transcode(inputPath, outputPath, Presets["discord"], ffCfg)
}

// Normalize converts audio to a standard mixing format (44.1kHz pcm_s16le mono WAV).
func Normalize(inputPath, outputPath string, ffCfg ffmpeg.Config) error {
	return Transcode(inputPath, outputPath, Presets["lossless"], ffCfg)
}

// ConcatAudio joins multiple audio files using ffmpeg's concat demuxer.
func ConcatAudio(inputs []string, outputPath string, ffCfg ffmpeg.Config) error {
	if len(inputs) == 0 {
		return fmt.Errorf("no input files to concatenate")
	}
	if len(inputs) == 1 {
		data, err := os.ReadFile(inputs[0])
		if err != nil {
			return err
		}
		return os.WriteFile(outputPath, data, 0644)
	}

	// Write concat list file
	listFile, err := os.CreateTemp("", "forge-concat-*.txt")
	if err != nil {
		return fmt.Errorf("creating concat list: %w", err)
	}
	defer os.Remove(listFile.Name())

	for _, input := range inputs {
		abs, _ := filepath.Abs(input)
		fmt.Fprintf(listFile, "file '%s'\n", abs)
	}
	listFile.Close()

	args := []string{
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", listFile.Name(),
		"-c", "copy",
		outputPath,
	}
	return ffmpeg.RunSilent(ffCfg, args...)
}

// DiscordFFmpegArgs returns ffmpeg args for a conservative MP3 export that
// Discord's inline attachment player handles reliably.
func DiscordFFmpegArgs(inputPath, outputPath string) []string {
	return Presets["discord"].FFmpegArgs(inputPath, outputPath)
}

// ListenPagePath returns the default sidecar HTML page path for an audio file.
func ListenPagePath(audioPath string) string {
	ext := filepath.Ext(audioPath)
	base := strings.TrimSuffix(audioPath, ext)
	return base + ".listen.html"
}

// WriteListenPage writes a self-contained HTML player with the audio embedded as a data URL.
func WriteListenPage(audioPath, title, text string) (string, error) {
	audioBytes, err := os.ReadFile(audioPath)
	if err != nil {
		return "", fmt.Errorf("reading audio for listen page: %w", err)
	}

	if title == "" {
		title = strings.TrimSuffix(filepath.Base(audioPath), filepath.Ext(audioPath))
	}

	data := struct {
		Title   string
		Text    string
		MIME    string
		Base64  string
		Format  string
		Audio   string
		HasText bool
	}{
		Title:   title,
		Text:    text,
		MIME:    audioMIME(audioPath),
		Base64:  base64.StdEncoding.EncodeToString(audioBytes),
		Format:  strings.TrimPrefix(strings.ToLower(filepath.Ext(audioPath)), "."),
		Audio:   filepath.Base(audioPath),
		HasText: strings.TrimSpace(text) != "",
	}

	const page = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <style>
    :root {
      color-scheme: light;
      --bg: #f5efe3;
      --card: #fffaf2;
      --ink: #1f1a17;
      --muted: #6b625d;
      --line: #d8ccbc;
      --accent: #a44d2d;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: Georgia, "Iowan Old Style", serif;
      background:
        radial-gradient(circle at top left, rgba(164,77,45,.14), transparent 28rem),
        linear-gradient(180deg, #f7f1e7, #efe4d3);
      color: var(--ink);
      min-height: 100vh;
      display: grid;
      place-items: center;
      padding: 24px;
    }
    main {
      width: min(720px, 100%);
      background: rgba(255,250,242,.94);
      border: 1px solid var(--line);
      border-radius: 20px;
      padding: 28px;
      box-shadow: 0 24px 80px rgba(63, 39, 28, .12);
    }
    h1 {
      margin: 0 0 10px;
      font-size: clamp(2rem, 4vw, 3rem);
      line-height: 1;
    }
    p {
      margin: 0 0 18px;
      color: var(--muted);
      font-size: 1rem;
      line-height: 1.5;
      white-space: pre-wrap;
    }
    .meta {
      display: flex;
      gap: 12px;
      flex-wrap: wrap;
      margin-bottom: 20px;
      color: var(--muted);
      font-size: .95rem;
    }
    .pill {
      border: 1px solid var(--line);
      border-radius: 999px;
      padding: 6px 10px;
      background: rgba(255,255,255,.5);
    }
    audio {
      width: 100%;
      margin: 8px 0 18px;
    }
    .hint {
      font-size: .95rem;
      color: var(--muted);
    }
    a { color: var(--accent); }
  </style>
</head>
<body>
  <main>
    <h1>{{.Title}}</h1>
    {{if .HasText}}<p>{{.Text}}</p>{{end}}
    <div class="meta">
      <span class="pill">Format: {{.Format}}</span>
      <span class="pill">Audio file: {{.Audio}}</span>
    </div>
    <audio controls preload="metadata">
      <source src="data:{{.MIME}};base64,{{.Base64}}" type="{{.MIME}}">
      Your browser does not support embedded audio playback.
    </audio>
    <div class="hint">This page is self-contained, so you can host this single HTML file anywhere and share the resulting link.</div>
  </main>
</body>
</html>
`

	tpl, err := template.New("listen").Parse(page)
	if err != nil {
		return "", fmt.Errorf("parse listen page template: %w", err)
	}

	outPath := ListenPagePath(audioPath)
	f, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("creating listen page: %w", err)
	}
	defer f.Close()

	if err := tpl.Execute(f, data); err != nil {
		return "", fmt.Errorf("writing listen page: %w", err)
	}

	return outPath, nil
}

func audioMIME(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".mp3":
		return "audio/mpeg"
	case ".ogg":
		return "audio/ogg"
	case ".wav":
		return "audio/wav"
	case ".m4a":
		return "audio/mp4"
	default:
		return "application/octet-stream"
	}
}
