package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/cyperx84/voice-forge/internal/analyzer"
	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/corpus"
	"github.com/cyperx84/voice-forge/internal/profile"
	"github.com/cyperx84/voice-forge/internal/skill"
	"github.com/cyperx84/voice-forge/internal/watch"
	"github.com/spf13/cobra"
)

var pipelineCmd = &cobra.Command{
	Use:   "pipeline",
	Short: "Run the full voice forge pipeline end-to-end",
	Long: `Runs the full pipeline in order:
  1. forge ingest — pick up any new files
  2. forge refresh — re-analyze if needed
  3. forge skill — update the OpenClaw skill

With --watch: run continuously (watch + periodic refresh + skill update).

Examples:
  forge pipeline
  forge pipeline --watch
  forge pipeline --skill-output ~/.openclaw/skills/cyperx-voice/`,
	RunE: func(cmd *cobra.Command, args []string) error {
		watchMode, _ := cmd.Flags().GetBool("watch")
		skillOutput, _ := cmd.Flags().GetString("skill-output")

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		if skillOutput == "" {
			skillOutput = cfg.SkillOutputDir()
		} else {
			skillOutput = config.ExpandPath(skillOutput)
		}

		if watchMode {
			return runContinuousPipeline(cfg, skillOutput)
		}

		return runOncePipeline(cfg, skillOutput)
	},
}

func runOncePipeline(cfg config.Config, skillOutput string) error {
	fmt.Println("=== Voice Forge Pipeline ===")

	// Step 1: Ingest — process any new .ogg files in the watch directory
	fmt.Println("[1/3] Ingesting new files...")
	watchDir := cfg.WatchDir()
	w := &watch.Watcher{
		Dir:            watchDir,
		WhisperCommand: cfg.Watch.WhisperCommand,
		WhisperModel:   cfg.Watch.WhisperModel,
		OpenAIAPIKey:   cfg.Watch.OpenAIAPIKey,
	}
	n, err := w.ProcessExisting()
	if err != nil {
		fmt.Printf("  warning: ingest error: %v\n", err)
	} else {
		fmt.Printf("  processed %d new file(s)\n", n)
	}

	// Step 2: Refresh — re-analyze if needed
	fmt.Println("\n[2/3] Checking if refresh needed...")
	paths := cfg.CorpusPaths()
	transcripts, err := corpus.ReadTranscripts(paths)
	if err != nil {
		return fmt.Errorf("reading transcripts: %w", err)
	}

	profileDir := cfg.ProfileDir()
	stylePath := filepath.Join(profileDir, "style.json")

	refreshed := false
	if len(transcripts) > 0 {
		shouldRefresh, reason := needsRefresh(stylePath, len(transcripts), cfg.Refresh)
		if shouldRefresh {
			fmt.Printf("  refreshing: %s\n", reason)
			prof, err := analyzer.Analyze(transcripts, cfg.LLM.Command, cfg.LLM.Args)
			if err != nil {
				return fmt.Errorf("analysis failed: %w", err)
			}
			if err := analyzer.SaveProfile(prof, profileDir); err != nil {
				return fmt.Errorf("saving profile: %w", err)
			}
			refreshed = true
			fmt.Printf("  profile updated: %d samples, %d words\n", prof.SampleCount, prof.TotalWords)
		} else {
			fmt.Printf("  skipped: %s\n", reason)
		}
	} else {
		fmt.Println("  no transcripts found")
	}

	// Step 3: Skill — update OpenClaw skill
	fmt.Println("\n[3/3] Updating OpenClaw skill...")
	p, err := profile.Load(stylePath)
	if err != nil {
		fmt.Printf("  skipped: no profile found (run 'forge analyze' first)\n")
	} else {
		if err := skill.Generate(p, skillOutput); err != nil {
			return fmt.Errorf("generating skill: %w", err)
		}
		fmt.Printf("  skill updated at %s\n", skillOutput)
	}

	// Summary
	fmt.Println("\n=== Pipeline Complete ===")
	fmt.Printf("  Ingested:  %d new file(s)\n", n)
	fmt.Printf("  Refreshed: %v\n", refreshed)
	fmt.Printf("  Skill:     %s\n", skillOutput)

	return nil
}

func runContinuousPipeline(cfg config.Config, skillOutput string) error {
	fmt.Println("=== Voice Forge Pipeline (continuous mode) ===")

	// Run once first
	if err := runOncePipeline(cfg, skillOutput); err != nil {
		fmt.Printf("initial pipeline error: %v\n", err)
	}

	// Parse refresh interval for periodic re-runs
	refreshInterval, err := time.ParseDuration(cfg.Refresh.MinInterval)
	if err != nil {
		refreshInterval = 24 * time.Hour
	}

	watchDir := cfg.WatchDir()
	pollInterval, err := time.ParseDuration(cfg.Watch.Interval)
	if err != nil {
		pollInterval = 30 * time.Second
	}

	fileWriteDelay, _ := time.ParseDuration(cfg.Watch.FileWriteDelay)

	w := &watch.Watcher{
		Dir:            watchDir,
		Interval:       pollInterval,
		FileWriteDelay: fileWriteDelay,
		WhisperCommand: cfg.Watch.WhisperCommand,
		WhisperModel:   cfg.Watch.WhisperModel,
		OpenAIAPIKey:   cfg.Watch.OpenAIAPIKey,
		OnIngest: func(path string) {
			fmt.Printf("  [auto] ingested: %s\n", filepath.Base(path))
		},
	}

	stop := make(chan struct{})
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sig)

	go func() {
		<-sig
		fmt.Println("\nshutting down pipeline...")
		close(stop)
	}()

	// Periodic refresh + skill update (ingest is handled by w.Run below)
	go func() {
		ticker := time.NewTicker(refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				fmt.Println("\n[periodic] running refresh + skill update...")
				if err := runRefreshAndSkill(cfg, skillOutput); err != nil {
					fmt.Printf("[periodic] error: %v\n", err)
				}
			}
		}
	}()

	return w.Run(stop)
}

// runRefreshAndSkill runs only the refresh + skill update steps (no ingest).
// Used by the periodic goroutine in continuous mode to avoid racing with w.Run().
func runRefreshAndSkill(cfg config.Config, skillOutput string) error {
	paths := cfg.CorpusPaths()
	transcripts, err := corpus.ReadTranscripts(paths)
	if err != nil {
		return fmt.Errorf("reading transcripts: %w", err)
	}

	profileDir := cfg.ProfileDir()
	stylePath := filepath.Join(profileDir, "style.json")

	if len(transcripts) > 0 {
		shouldRefresh, reason := needsRefresh(stylePath, len(transcripts), cfg.Refresh)
		if shouldRefresh {
			fmt.Printf("  refreshing: %s\n", reason)
			prof, err := analyzer.Analyze(transcripts, cfg.LLM.Command, cfg.LLM.Args)
			if err != nil {
				return fmt.Errorf("analysis failed: %w", err)
			}
			if err := analyzer.SaveProfile(prof, profileDir); err != nil {
				return fmt.Errorf("saving profile: %w", err)
			}
			fmt.Printf("  profile updated: %d samples, %d words\n", prof.SampleCount, prof.TotalWords)
		} else {
			fmt.Printf("  skipped refresh: %s\n", reason)
		}
	}

	p, err := profile.Load(stylePath)
	if err != nil {
		fmt.Printf("  skipped skill: no profile found\n")
		return nil
	}
	if err := skill.Generate(p, skillOutput); err != nil {
		return fmt.Errorf("generating skill: %w", err)
	}
	fmt.Printf("  skill updated at %s\n", skillOutput)
	return nil
}

func init() {
	pipelineCmd.Flags().Bool("watch", false, "run continuously (watch + periodic refresh + skill update)")
	pipelineCmd.Flags().String("skill-output", "", "output directory for skill files (default from config)")
	rootCmd.AddCommand(pipelineCmd)
}
