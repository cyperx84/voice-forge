package watch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cyperx84/voice-forge/internal/ffmpeg"
	"github.com/fsnotify/fsnotify"
)

// Watcher monitors a directory for new .ogg files and auto-ingests them.
type Watcher struct {
	Dir            string
	Interval       time.Duration
	FileWriteDelay time.Duration // delay after file event before processing (default 500ms)
	WhisperCommand string
	WhisperModel   string
	OpenAIAPIKey   string
	FFmpegCfg      ffmpeg.Config     // ffmpeg resource limits
	OnIngest       func(path string) // callback after successful ingest
	mu             sync.Mutex        // guards concurrent ProcessExisting calls
}

// DryRun scans the directory and prints unprocessed .ogg files without processing them.
func DryRun(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Directory does not exist: %s\n", dir)
			return nil
		}
		return err
	}

	unprocessed := 0
	skipped := 0
	for _, e := range entries {
		if e.IsDir() || !isOggFile(e.Name()) {
			continue
		}
		oggPath := filepath.Join(dir, e.Name())
		base := strings.TrimSuffix(oggPath, filepath.Ext(oggPath))
		if _, err := os.Stat(base + ".txt"); err == nil {
			skipped++
			continue
		}
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		fmt.Printf("  [new] %s (%d bytes)\n", e.Name(), size)
		unprocessed++
	}

	fmt.Printf("\nSummary: %d unprocessed, %d already transcribed\n", unprocessed, skipped)
	return nil
}

// ProcessExisting scans the directory for unprocessed .ogg files and processes them.
func (w *Watcher) ProcessExisting() (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	entries, err := os.ReadDir(w.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	processed := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !isOggFile(name) {
			continue
		}
		oggPath := filepath.Join(w.Dir, name)
		if w.hasTranscript(oggPath) {
			continue
		}
		if err := w.ingest(oggPath); err != nil {
			log.Printf("error processing %s: %v", name, err)
			continue
		}
		processed++
	}
	return processed, nil
}

// Run starts the file watcher. It blocks until the context is cancelled or an error occurs.
func (w *Watcher) Run(stop <-chan struct{}) error {
	if err := os.MkdirAll(w.Dir, 0755); err != nil {
		return fmt.Errorf("creating watch directory: %w", err)
	}

	// Process any existing unprocessed files first
	n, err := w.ProcessExisting()
	if err != nil {
		log.Printf("warning: initial scan error: %v", err)
	}
	if n > 0 {
		log.Printf("processed %d existing file(s)", n)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("creating watcher: %w", err)
	}
	defer watcher.Close()

	if err := watcher.Add(w.Dir); err != nil {
		return fmt.Errorf("watching directory: %w", err)
	}

	log.Printf("watching %s for new .ogg files (poll interval: %s)", w.Dir, w.Interval)

	// Use a ticker as a fallback poll in case fsnotify misses events (e.g. NFS)
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			log.Println("watcher stopped")
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Op&(fsnotify.Create|fsnotify.Write) == 0 {
				continue
			}
			if !isOggFile(event.Name) {
				continue
			}
			// Small delay to let the file finish writing
			delay := w.FileWriteDelay
			if delay == 0 {
				delay = 500 * time.Millisecond
			}
			time.Sleep(delay)
			if w.hasTranscript(event.Name) {
				continue
			}
			if err := w.ingest(event.Name); err != nil {
				log.Printf("error processing %s: %v", filepath.Base(event.Name), err)
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			log.Printf("watcher error: %v", err)

		case <-ticker.C:
			n, err := w.ProcessExisting()
			if err != nil {
				log.Printf("poll scan error: %v", err)
			}
			if n > 0 {
				log.Printf("poll: processed %d file(s)", n)
			}
		}
	}
}

// ingest converts an .ogg file to WAV, transcribes it, and saves the transcript.
func (w *Watcher) ingest(oggPath string) error {
	base := strings.TrimSuffix(filepath.Base(oggPath), filepath.Ext(oggPath))
	dir := filepath.Dir(oggPath)

	// Step 1: Convert to WAV (30s timeout)
	wavPath := filepath.Join(dir, base+".wav")
	if _, err := os.Stat(wavPath); os.IsNotExist(err) {
		log.Printf("converting %s -> %s.wav", filepath.Base(oggPath), base)
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := ffmpeg.CommandContext(ctx, w.FFmpegCfg, "-i", oggPath, "-ar", "16000", "-ac", "1", wavPath)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ffmpeg conversion: %w", err)
		}
	}

	// Step 2: Transcribe
	txtPath := filepath.Join(dir, base+".txt")
	if _, err := os.Stat(txtPath); err == nil {
		log.Printf("skip (transcript exists): %s", base)
		if w.OnIngest != nil {
			w.OnIngest(oggPath)
		}
		return nil
	}

	log.Printf("transcribing %s", base)
	transcript, err := w.transcribe(wavPath)
	if err != nil {
		return fmt.Errorf("transcription: %w", err)
	}

	// Step 3: Save transcript
	if err := os.WriteFile(txtPath, []byte(strings.TrimSpace(transcript)), 0644); err != nil {
		return fmt.Errorf("saving transcript: %w", err)
	}

	log.Printf("ingested: %s (%d chars)", base, len(transcript))

	if w.OnIngest != nil {
		w.OnIngest(oggPath)
	}
	return nil
}

// transcribe tries whisper backends in priority order:
// 1. Configured whisper command (mlx_whisper, whisper-cli, etc.)
// 2. mlx_whisper (Apple Silicon native)
// 3. whisper-cli / whisper-cpp
// 4. OpenAI Whisper API (if API key configured)
func (w *Watcher) transcribe(audioPath string) (string, error) {
	// Try configured command first
	if w.WhisperCommand != "" {
		if out, err := w.runWhisperCommand(w.WhisperCommand, audioPath); err == nil {
			return out, nil
		} else {
			log.Printf("whisper command %q failed: %v, trying fallbacks...", w.WhisperCommand, err)
		}
	}

	// Fallback chain
	fallbacks := []string{"mlx_whisper", "whisper-cli", "whisper-cpp"}
	for _, cmd := range fallbacks {
		if cmd == w.WhisperCommand {
			continue // already tried
		}
		if _, err := exec.LookPath(cmd); err != nil {
			continue // not installed
		}
		if out, err := w.runWhisperCommand(cmd, audioPath); err == nil {
			return out, nil
		}
	}

	// Final fallback: OpenAI Whisper API
	if w.OpenAIAPIKey != "" {
		log.Printf("trying OpenAI Whisper API...")
		return w.transcribeOpenAI(audioPath)
	}

	return "", fmt.Errorf("no working whisper backend found (tried %q and fallbacks); set openai_api_key for API fallback", w.WhisperCommand)
}

// runWhisperCommand runs a local whisper binary with the given audio file.
func (w *Watcher) runWhisperCommand(command, audioPath string) (string, error) {
	args := []string{audioPath}
	if w.WhisperModel != "" {
		args = append(args, "--model", w.WhisperModel)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// transcribeOpenAI calls the OpenAI Whisper API.
func (w *Watcher) transcribeOpenAI(audioPath string) (string, error) {
	f, err := os.Open(audioPath)
	if err != nil {
		return "", fmt.Errorf("opening audio file: %w", err)
	}
	defer f.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filepath.Base(audioPath))
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", err
	}
	if err := writer.WriteField("model", "whisper-1"); err != nil {
		return "", err
	}
	writer.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/audio/transcriptions", &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+w.OpenAIAPIKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("OpenAI API request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("OpenAI API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parsing OpenAI response: %w", err)
	}

	return result.Text, nil
}

// hasTranscript checks if a .txt transcript already exists for the given audio file.
func (w *Watcher) hasTranscript(audioPath string) bool {
	base := strings.TrimSuffix(audioPath, filepath.Ext(audioPath))
	_, err := os.Stat(base + ".txt")
	return err == nil
}

func isOggFile(name string) bool {
	return strings.ToLower(filepath.Ext(name)) == ".ogg"
}

// CountTranscripts returns the number of .txt files in the watch directory.
func CountTranscripts(dir string) int {
	files, err := filepath.Glob(filepath.Join(dir, "*.txt"))
	if err != nil {
		return 0
	}
	return len(files)
}
