package ffmpeg

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Config controls ffmpeg resource usage.
type Config struct {
	Threads int `toml:"threads"` // max threads for ffmpeg (0 = ffmpeg default)
	Nice    int `toml:"nice"`    // nice value on Unix (0 = no change)
}

// DefaultConfig returns conservative defaults that won't saturate the CPU.
func DefaultConfig() Config {
	return Config{
		Threads: 4,
		Nice:    10,
	}
}

// PrependArgs returns args with -threads prepended if configured.
func (c Config) PrependArgs(args []string) []string {
	if c.Threads > 0 {
		return append([]string{"-threads", fmt.Sprintf("%d", c.Threads)}, args...)
	}
	return args
}

// wrapNice returns the command name and args wrapped with nice(1) if configured.
func wrapNice(nice int, name string, args []string) (string, []string) {
	if nice <= 0 || runtime.GOOS == "windows" {
		return name, args
	}
	return "nice", append([]string{"-n", fmt.Sprintf("%d", nice), name}, args...)
}

// Command returns a configured exec.Cmd for ffmpeg with thread and nice limits.
func Command(cfg Config, args ...string) *exec.Cmd {
	finalArgs := cfg.PrependArgs(args)
	name, wrappedArgs := wrapNice(cfg.Nice, "ffmpeg", finalArgs)
	return exec.Command(name, wrappedArgs...)
}

// CommandContext returns a configured exec.Cmd for ffmpeg with context support.
func CommandContext(ctx context.Context, cfg Config, args ...string) *exec.Cmd {
	finalArgs := cfg.PrependArgs(args)
	name, wrappedArgs := wrapNice(cfg.Nice, "ffmpeg", finalArgs)
	return exec.CommandContext(ctx, name, wrappedArgs...)
}

// Run executes ffmpeg with resource limits applied and returns combined output.
func Run(cfg Config, args ...string) ([]byte, error) {
	cmd := Command(cfg, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("ffmpeg failed: %w\noutput: %s", err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

// RunSilent executes ffmpeg, returning only the error (if any).
func RunSilent(cfg Config, args ...string) error {
	_, err := Run(cfg, args...)
	return err
}

// RunContext executes ffmpeg with a context for cancellation/timeout.
func RunContext(ctx context.Context, cfg Config, args ...string) ([]byte, error) {
	cmd := CommandContext(ctx, cfg, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("ffmpeg failed: %w\noutput: %s", err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

// ProbeCommand returns a configured exec.Cmd for ffprobe with nice limits.
func ProbeCommand(ctx context.Context, cfg Config, args ...string) *exec.Cmd {
	name, wrappedArgs := wrapNice(cfg.Nice, "ffprobe", args)
	return exec.CommandContext(ctx, name, wrappedArgs...)
}
