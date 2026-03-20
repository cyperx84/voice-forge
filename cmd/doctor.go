package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/cyperx84/voice-forge/internal/config"
	"github.com/cyperx84/voice-forge/internal/corpus"
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

		// Check Chatterbox
		checkPython("Chatterbox", "chatterbox", "pip3 install chatterbox-tts")

		// Check F5-TTS
		checkPython("F5-TTS", "f5_tts", "pip3 install f5-tts")

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

		// Check corpus size on disk
		corpusRoot := cfg.CorpusRoot()
		if info, err := os.Stat(corpusRoot); err == nil && info.IsDir() {
			size := dirSize(corpusRoot)
			fmt.Printf("  ℹ️  Disk: corpus is %s\n", formatSize(size))
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

func checkPython(label, module, installHint string) {
	cmd := exec.Command("python3", "-c", "import "+module)
	if cmd.Run() == nil {
		fmt.Printf("  ✅ %s: installed\n", label)
	} else {
		fmt.Printf("  ❌ %s: not installed (%s)\n", label, installHint)
	}
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
