class VoiceForge < Formula
  desc "Personal voice corpus management, style extraction, and character voice CLI"
  homepage "https://github.com/cyperx84/voice-forge"
  url "https://github.com/cyperx84/voice-forge/archive/refs/tags/v0.5.0.tar.gz"
  sha256 "PLACEHOLDER_SHA256"
  license "MIT"
  head "https://github.com/cyperx84/voice-forge.git", branch: "main"

  depends_on "go" => :build
  depends_on "ffmpeg"

  def install
    system "go", "build", *std_go_args(ldflags: "-s -w"), "-o", bin/"forge", "."

    generate_completions_from_executable(bin/"forge", "completion")
  end

  def caveats
    <<~EOS
      Voice Forge is installed as `forge`.

      Optional dependencies for TTS backends:
        pip3 install chatterbox-tts   # Chatterbox Turbo (recommended)
        pip3 install f5-tts           # F5-TTS

      Optional dependency for transcription:
        brew install whisper-cli

      Get started:
        forge doctor          # check environment
        forge backends        # list TTS backends
        forge ingest-bulk --help
    EOS
  end

  test do
    assert_match "Voice Forge", shell_output("#{bin}/forge --help")
    assert_match "doctor", shell_output("#{bin}/forge --help")
  end
end
