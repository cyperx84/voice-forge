package ingest

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/cyperx84/voice-forge/internal/corpus"
	"github.com/google/uuid"
)

// VideoOptions configures video ingestion.
type VideoOptions struct {
	Source          string
	TranscriptOnly bool
	WhisperCommand string
	WhisperModel   string
	KeyframeInterval int // seconds between keyframe extractions
	Tags           []string
}

// IngestVideoFile processes a video file: copies it, extracts transcript, optionally keyframes.
func IngestVideoFile(db *corpus.DB, corpusRoot, filePath string, opts VideoOptions) (*corpus.Item, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("stat file: %w", err)
	}

	source := opts.Source
	if source == "" {
		source = "local"
	}

	id := uuid.New().String()
	ext := filepath.Ext(filePath)
	destDir := filepath.Join(corpusRoot, "video")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, err
	}

	// Copy video file
	destPath := filepath.Join(destDir, id+ext)
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("reading video: %w", err)
	}
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return nil, err
	}

	// Get duration from ffprobe
	duration := getVideoDuration(filePath)

	// Extract transcript via whisper
	var transcript string
	whisperCmd := opts.WhisperCommand
	if whisperCmd == "" {
		whisperCmd = "whisper-cli"
	}
	// Extract audio track to WAV first
	wavPath := filepath.Join(destDir, id+".wav")
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	ffCmd := exec.CommandContext(ctx, "ffmpeg", "-i", filePath, "-ar", "16000", "-ac", "1", "-vn", wavPath)
	if err := ffCmd.Run(); err == nil {
		// Try transcription
		tCtx, tCancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer tCancel()
		args := []string{wavPath}
		if opts.WhisperModel != "" {
			args = append(args, "--model", opts.WhisperModel)
		}
		wCmd := exec.CommandContext(tCtx, whisperCmd, args...)
		if out, err := wCmd.Output(); err == nil {
			transcript = strings.TrimSpace(string(out))
		}
		os.Remove(wavPath)
	}

	// Extract keyframes (unless transcript-only)
	if !opts.TranscriptOnly {
		interval := opts.KeyframeInterval
		if interval <= 0 {
			interval = 10
		}
		kfDir := filepath.Join(destDir, id+"_keyframes")
		os.MkdirAll(kfDir, 0755)
		kfCtx, kfCancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer kfCancel()
		kfCmd := exec.CommandContext(kfCtx, "ffmpeg", "-i", filePath,
			"-vf", fmt.Sprintf("fps=1/%d", interval),
			filepath.Join(kfDir, "frame_%04d.jpg"))
		kfCmd.Run() // best-effort
	}

	metadata := map[string]string{
		"original_name": filepath.Base(filePath),
	}
	if duration > 0 {
		metadata["duration"] = fmt.Sprintf("%.1fs", duration)
	}

	item := &corpus.Item{
		ID:              id,
		Type:            corpus.TypeVideo,
		Source:          source,
		CreatedAt:       info.ModTime().Format(time.RFC3339),
		IngestedAt:      time.Now().Format(time.RFC3339),
		Path:            filepath.Join("video", id+ext),
		Transcript:      transcript,
		Tags:            opts.Tags,
		Metadata:        metadata,
		WordCount:       len(strings.Fields(transcript)),
		DurationSeconds: duration,
		FileSize:        info.Size(),
	}

	if err := db.Insert(item); err != nil {
		return nil, fmt.Errorf("inserting into db: %w", err)
	}

	return item, nil
}

func getVideoDuration(path string) float64 {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "ffprobe", "-v", "quiet", "-show_entries", "format=duration", "-of", "default=noprint_wrappers=1:nokey=1", path)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	var dur float64
	fmt.Sscanf(strings.TrimSpace(string(out)), "%f", &dur)
	return dur
}
