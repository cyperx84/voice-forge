package live

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cyperx84/voice-forge/internal/config"
)

var (
	mu        sync.Mutex
	proc      *os.Process
	startedAt time.Time
)

// BotDir returns the path to the gemini-live-discord bot directory.
func BotDir() string {
	return config.ExpandPath("~/.forge/live")
}

// SetupBot ensures the bot directory exists.
func SetupBot() error {
	dir := BotDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating bot directory: %w", err)
	}
	botMjs := filepath.Join(dir, "bot.mjs")
	if _, err := os.Stat(botMjs); os.IsNotExist(err) {
		return fmt.Errorf("bot.mjs not found at %s — clone gemini-live-discord into %s", botMjs, dir)
	}
	return nil
}

// writeEnv generates a .env file in the bot directory from the live config.
func writeEnv(cfg config.LiveConfig) error {
	dir := BotDir()
	envPath := filepath.Join(dir, ".env")

	apiKey := cfg.GeminiAPIKey
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}
	discordToken := cfg.Discord.Token
	if discordToken == "" {
		discordToken = os.Getenv("DISCORD_TOKEN")
	}

	var sb strings.Builder
	writeLine := func(k, v string) {
		if v != "" {
			fmt.Fprintf(&sb, "%s=%s\n", k, v)
		}
	}

	writeLine("GEMINI_API_KEY", apiKey)
	writeLine("GEMINI_MODEL", cfg.Model)
	writeLine("VOICE_NAME", cfg.Voice)
	writeLine("LANGUAGE_CODE", cfg.Language)
	writeLine("DISCORD_TOKEN", discordToken)
	writeLine("VOICE_CHANNEL_ID", cfg.Discord.VoiceChannel)
	writeLine("GUILD_ID", cfg.Discord.Guild)
	writeLine("TARGET_USER_ID", cfg.Discord.TargetUser)
	writeLine("VAD_START_SENSITIVITY", cfg.VAD.StartSensitivity)
	writeLine("VAD_END_SENSITIVITY", cfg.VAD.EndSensitivity)
	if cfg.VAD.PrefixPaddingMs > 0 {
		writeLine("VAD_PREFIX_PADDING_MS", strconv.Itoa(cfg.VAD.PrefixPaddingMs))
	}
	if cfg.VAD.SilenceDurationMs > 0 {
		writeLine("VAD_SILENCE_DURATION_MS", strconv.Itoa(cfg.VAD.SilenceDurationMs))
	}
	writeLine("SYSTEM_PROMPT", cfg.SystemPrompt)
	writeLine("GREETING_PROMPT", cfg.GreetingPrompt)

	return os.WriteFile(envPath, []byte(sb.String()), 0600)
}

// Start generates the .env file and spawns the Node.js bot as a subprocess.
func Start(cfg config.LiveConfig) error {
	mu.Lock()
	defer mu.Unlock()

	if proc != nil {
		return fmt.Errorf("live session already running (pid %d)", proc.Pid)
	}

	if err := SetupBot(); err != nil {
		return err
	}

	if err := writeEnv(cfg); err != nil {
		return fmt.Errorf("writing .env: %w", err)
	}

	dir := BotDir()
	cmd := exec.Command("node", "bot.mjs")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting bot: %w", err)
	}

	proc = cmd.Process
	startedAt = time.Now()

	// Reap the process in the background so it doesn't become a zombie.
	go func() {
		_ = cmd.Wait()
		mu.Lock()
		proc = nil
		mu.Unlock()
	}()

	return nil
}

// Stop sends SIGTERM to the running bot process.
func Stop() error {
	mu.Lock()
	defer mu.Unlock()

	if proc == nil {
		return fmt.Errorf("no live session is running")
	}

	if err := proc.Signal(os.Interrupt); err != nil {
		// Fall back to Kill if Interrupt is not supported.
		if killErr := proc.Kill(); killErr != nil {
			return fmt.Errorf("stopping bot: %w", killErr)
		}
	}

	proc = nil
	return nil
}

// StatusInfo holds the current bot status.
type StatusInfo struct {
	Running bool
	PID     int
	Uptime  time.Duration
}

// Status returns the current state of the bot subprocess.
func Status() StatusInfo {
	mu.Lock()
	defer mu.Unlock()

	if proc == nil {
		return StatusInfo{Running: false}
	}

	return StatusInfo{
		Running: true,
		PID:     proc.Pid,
		Uptime:  time.Since(startedAt),
	}
}
