package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/watch"
	"github.com/spf13/cobra"
)

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Monitor for new voice messages and auto-ingest them",
	Long: `Watches the voice-corpus directory for new .ogg files.
When a new file appears:
  1. Convert to WAV (ffmpeg)
  2. Transcribe (whisper)
  3. Save transcript as .txt alongside the audio
  4. Add to corpus index

Runs as a long-lived process. Designed for OpenClaw cron or systemd.

Examples:
  forge watch
  forge watch --dir ~/.openclaw/workspace/voice-corpus/
  forge watch --interval 30s`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		dir, _ := cmd.Flags().GetString("dir")
		if dir == "" {
			dir = cfg.WatchDir()
		} else {
			dir = config.ExpandPath(dir)
		}

		dryRun, _ := cmd.Flags().GetBool("dry-run")
		if dryRun {
			return watch.DryRun(dir)
		}

		intervalStr, _ := cmd.Flags().GetString("interval")
		if intervalStr == "" {
			intervalStr = cfg.Watch.Interval
		}
		interval, err := time.ParseDuration(intervalStr)
		if err != nil {
			return fmt.Errorf("invalid interval %q: %w", intervalStr, err)
		}

		whisperCmd := cfg.Watch.WhisperCommand
		if whisperCmd == "" {
			whisperCmd = "whisper-cli"
		}

		w := &watch.Watcher{
			Dir:            dir,
			Interval:       interval,
			WhisperCommand: whisperCmd,
			WhisperModel:   cfg.Watch.WhisperModel,
			OpenAIAPIKey:   cfg.Watch.OpenAIAPIKey,
		}

		stop := make(chan struct{})
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

		go func() {
			<-sig
			fmt.Println("\nshutting down watcher...")
			close(stop)
		}()

		return w.Run(stop)
	},
}

func init() {
	watchCmd.Flags().String("dir", "", "directory to watch (default from config)")
	watchCmd.Flags().String("interval", "", "poll interval (default from config)")
	watchCmd.Flags().Bool("dry-run", false, "show unprocessed files without processing them")
	rootCmd.AddCommand(watchCmd)
}
