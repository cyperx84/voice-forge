package ffmpeg

import (
	"strings"
	"testing"
)

func TestPrependArgs_WithThreads(t *testing.T) {
	cfg := Config{Threads: 4}
	got := cfg.PrependArgs([]string{"-y", "-i", "in.wav", "out.mp3"})
	joined := strings.Join(got, " ")
	if !strings.HasPrefix(joined, "-threads 4") {
		t.Fatalf("expected args to start with -threads 4, got %q", joined)
	}
	if !strings.Contains(joined, "-y -i in.wav out.mp3") {
		t.Fatalf("expected original args preserved, got %q", joined)
	}
}

func TestPrependArgs_ZeroThreads(t *testing.T) {
	cfg := Config{Threads: 0}
	orig := []string{"-y", "-i", "in.wav", "out.mp3"}
	got := cfg.PrependArgs(orig)
	if len(got) != len(orig) {
		t.Fatalf("expected no extra args when Threads=0, got %v", got)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Threads != 4 {
		t.Fatalf("default threads = %d, want 4", cfg.Threads)
	}
	if cfg.Nice != 10 {
		t.Fatalf("default nice = %d, want 10", cfg.Nice)
	}
}

func TestCommand_IncludesThreads(t *testing.T) {
	cfg := Config{Threads: 2, Nice: 0}
	cmd := Command(cfg, "-y", "-i", "in.wav", "out.mp3")
	args := strings.Join(cmd.Args, " ")
	if !strings.Contains(args, "-threads 2") {
		t.Fatalf("expected -threads 2 in args, got %q", args)
	}
}

func TestCommand_WithNice(t *testing.T) {
	cfg := Config{Threads: 2, Nice: 15}
	cmd := Command(cfg, "-y", "-i", "in.wav", "out.mp3")
	// On non-Windows, should be wrapped with nice
	if cmd.Path != "" && strings.Contains(strings.Join(cmd.Args, " "), "nice") {
		args := strings.Join(cmd.Args, " ")
		if !strings.Contains(args, "-n 15") {
			t.Fatalf("expected nice -n 15 in args, got %q", args)
		}
		if !strings.Contains(args, "-threads 2") {
			t.Fatalf("expected -threads 2 in args, got %q", args)
		}
	}
}
