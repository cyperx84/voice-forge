package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/corpus"
	"github.com/cyperx84/voice-forge/internal/tts"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check entire environment and dependencies",
	Long:  `Runs diagnostic checks on config, corpus, tools, and TTS backends.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Voice Forge Doctor")
		fmt.Println()

		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("  ❌ Config: failed to load (%v)\n", err)
		} else {
			fmt.Printf("  ✅ Config: %s\n", config.ConfigPath())
		}

		// Check corpus DB
		dbPath := cfg.CorpusDBPath()
		if info, err := os.Stat(dbPath); err == nil {
			db, dbErr := corpus.OpenDB(dbPath)
			if dbErr == nil {
				count, _ := db.Count("")
				db.Close()
				fmt.Printf("  ✅ Corpus DB: %s (%d items, %s)\n", dbPath, count, formatSize(info.Size()))
			} else {
				fmt.Printf("  ❌ Corpus DB: %s (error: %v)\n", dbPath, dbErr)
			}
		} else {
			fmt.Printf("  ⚠️  Corpus DB: not found at %s (will be created on first ingest)\n", dbPath)
		}

		// Check style profile
		profilePath := cfg.ProfileDir() + "/style.json"
		if _, err := os.Stat(profilePath); err == nil {
			fmt.Printf("  ✅ Style profile: %s\n", profilePath)
		} else {
			fmt.Printf("  ⚠️  Style profile: not found (run 'forge analyze' to create)\n")
		}

		// Check ffmpeg
		checkTool("ffmpeg", "ffmpeg", "brew install ffmpeg")

		initBackends(cfg)

		// Check Chatterbox / F5 / Toolkit-backed runtimes using the same resolution path as forge speak.
		checkBackend("Chatterbox", "chatterbox", "pip3 install chatterbox-tts")

		// Check F5-TTS
		checkBackend("F5-TTS", "f5-tts", "pip3 install f5-tts")

		// Check Whisper
		checkTool("Whisper", cfg.Watch.WhisperCommand, "brew install whisper-cli")

		// Check ElevenLabs
		apiKey := cfg.TTS.ElevenLabs.APIKey
		if apiKey == "" {
			apiKey = os.Getenv("ELEVENLABS_API_KEY")
		}
		if apiKey != "" {
			fmt.Printf("  ✅ ElevenLabs: API key configured\n")
		} else {
			fmt.Printf("  ⚠️  ElevenLabs: no API key (set in ~/.forge/config.toml or ELEVENLABS_API_KEY)\n")
		}

		// Check corpus storage footprint (db + managed corpus root + configured source dirs).
		sizes := corpusFootprint(cfg)
		var parts []string
		if sizes.db > 0 {
			parts = append(parts, fmt.Sprintf("db %s", formatSize(sizes.db)))
		}
		if sizes.managed > 0 {
			parts = append(parts, fmt.Sprintf("managed %s", formatSize(sizes.managed)))
		}
		if sizes.sources > 0 {
			parts = append(parts, fmt.Sprintf("sources %s", formatSize(sizes.sources)))
		}
		if len(parts) > 0 {
			fmt.Printf("  ℹ️  Disk: %s (total %s)\n", strings.Join(parts, ", "), formatSize(sizes.total()))
		}

		fmt.Println()
		return nil
	},
}

func checkTool(label, command, installHint string) {
	if command == "" {
		fmt.Printf("  ⚠️  %s: not configured\n", label)
		return
	}
	_, err := exec.LookPath(command)
	if err == nil {
		// Try to get version
		out, verr := exec.Command(command, "-version").CombinedOutput()
		if verr == nil && len(out) > 0 {
			version := firstLine(string(out))
			if len(version) > 60 {
				version = version[:60]
			}
			fmt.Printf("  ✅ %s: %s\n", label, version)
		} else {
			fmt.Printf("  ✅ %s: installed\n", label)
		}
	} else {
		fmt.Printf("  ❌ %s: not installed (%s)\n", label, installHint)
	}
}

func checkBackend(label, backendName, installHint string) {
	b, err := tts.Get(backendName)
	if err != nil {
		fmt.Printf("  ❌ %s: backend not registered (%v)\n", label, err)
		return
	}
	if b.Available() {
		fmt.Printf("  ✅ %s: available\n", label)
		return
	}
	setup := b.Setup()
	if setup != nil {
		fmt.Printf("  ❌ %s: not available (%s)\n", label, firstLine(setup.Error()))
		return
	}
	fmt.Printf("  ❌ %s: not available (%s)\n", label, installHint)
}

type footprint struct {
	db      int64
	managed int64
	sources int64
}

func (f footprint) total() int64 { return f.db + f.managed + f.sources }

func corpusFootprint(cfg config.Config) footprint {
	var fp footprint
	if info, err := os.Stat(cfg.CorpusDBPath()); err == nil && !info.IsDir() {
		fp.db = info.Size()
	}
	if info, err := os.Stat(cfg.CorpusRoot()); err == nil && info.IsDir() {
		fp.managed = dirSize(cfg.CorpusRoot())
	}
	seen := map[string]bool{}
	for _, p := range cfg.CorpusPaths() {
		p = config.ExpandPath(p)
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			fp.sources += dirSize(p)
		}
	}
	return fp
}

func firstLine(s string) string {
	for i, c := range s {
		if c == '\n' || c == '\r' {
			return s[:i]
		}
	}
	return s
}

func dirSize(path string) int64 {
	var size int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		size += info.Size()
		return nil
	})
	return size
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
