package tts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ElevenLabsBackend calls the ElevenLabs REST API for TTS.
type ElevenLabsBackend struct {
	APIKey string
}

const elevenLabsBaseURL = "https://api.elevenlabs.io/v1"

var elevenLabsHTTPClient = &http.Client{Timeout: 30 * time.Second}

func (e *ElevenLabsBackend) Name() string { return "elevenlabs" }

func (e *ElevenLabsBackend) NativeFormat() AudioFormat {
	return AudioFormat{SampleRate: 44100, Channels: 1, Codec: "mp3", Container: "mp3"}
}

func (e *ElevenLabsBackend) Available() bool {
	return e.APIKey != ""
}

func (e *ElevenLabsBackend) Setup() error {
	if e.Available() {
		return nil
	}
	return fmt.Errorf("ElevenLabs API key not configured — set it in ~/.forge/config.toml under [tts.elevenlabs] or env ELEVENLABS_API_KEY")
}

func (e *ElevenLabsBackend) Speak(text string, opts SpeakOpts) ([]byte, error) {
	if !e.Available() {
		return nil, e.Setup()
	}

	voice := opts.Voice
	if voice == "" {
		voice = "21m00Tcm4TlvDq8ikWAM" // default ElevenLabs voice (Rachel)
	}

	url := fmt.Sprintf("%s/text-to-speech/%s", elevenLabsBaseURL, voice)

	body := map[string]interface{}{
		"text":     text,
		"model_id": "eleven_monolingual_v1",
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("xi-api-key", e.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "audio/mpeg")

	resp, err := elevenLabsHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ElevenLabs API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ElevenLabs API error (HTTP %d): %s", resp.StatusCode, errBody)
	}

	audio, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	return audio, nil
}

func (e *ElevenLabsBackend) Clone(samples []string, name string) error {
	if !e.Available() {
		return e.Setup()
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	if err := w.WriteField("name", name); err != nil {
		return fmt.Errorf("writing name field: %w", err)
	}
	if err := w.WriteField("description", "Voice cloned via Voice Forge"); err != nil {
		return fmt.Errorf("writing description field: %w", err)
	}

	allowedAudioExts := map[string]bool{
		".mp3": true, ".wav": true, ".ogg": true, ".m4a": true, ".flac": true, ".webm": true,
	}
	for _, sample := range samples {
		ext := strings.ToLower(filepath.Ext(sample))
		if !allowedAudioExts[ext] {
			return fmt.Errorf("unsupported audio format %q for sample %s (allowed: mp3, wav, ogg, m4a, flac, webm)", ext, sample)
		}
		f, err := os.Open(sample)
		if err != nil {
			return fmt.Errorf("opening sample %s: %w", sample, err)
		}
		part, err := w.CreateFormFile("files", filepath.Base(sample))
		if err != nil {
			f.Close()
			return fmt.Errorf("creating form file: %w", err)
		}
		if _, err := io.Copy(part, f); err != nil {
			f.Close()
			return fmt.Errorf("copying sample data: %w", err)
		}
		f.Close()
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("closing multipart writer: %w", err)
	}

	url := fmt.Sprintf("%s/voices/add", elevenLabsBaseURL)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("xi-api-key", e.APIKey)
	req.Header.Set("Content-Type", w.FormDataContentType())

	resp, err := elevenLabsHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("ElevenLabs clone request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ElevenLabs clone error (HTTP %d): %s", resp.StatusCode, errBody)
	}

	return nil
}
