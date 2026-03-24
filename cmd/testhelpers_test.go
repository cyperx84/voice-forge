package cmd

import "github.com/cyperx84/voice-forge/internal/config"

func testConfig() config.Config {
	cfg := config.DefaultConfig()
	cfg.TTS.TTSToolkit.Path = "/nonexistent"
	return cfg
}
