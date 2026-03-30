#!/usr/bin/env bash
set -euo pipefail

FORGE_HOME="${FORGE_HOME:-$HOME/.forge}"
CHATTERBOX_DIR="${CHATTERBOX_DIR:-$FORGE_HOME/venvs/chatterbox}"
F5_DIR="${F5_DIR:-$FORGE_HOME/venvs/f5-tts}"
PYTHON_BIN="${PYTHON_BIN:-/opt/homebrew/bin/python3.12}"

mkdir -p "$FORGE_HOME/venvs"

create_venv() {
  local dir="$1"
  "$PYTHON_BIN" -m venv "$dir"
  "$dir/bin/python3" -m pip install --upgrade pip setuptools wheel
}

if [ ! -x "$CHATTERBOX_DIR/bin/python3" ]; then
  create_venv "$CHATTERBOX_DIR"
fi
if [ ! -x "$F5_DIR/bin/python3" ]; then
  create_venv "$F5_DIR"
fi

echo "Installing Chatterbox into $CHATTERBOX_DIR"
"$CHATTERBOX_DIR/bin/python3" -m pip install chatterbox-tts

echo "Installing F5-TTS into $F5_DIR"
"$F5_DIR/bin/python3" -m pip install f5-tts

echo
echo "Runtime setup complete. Validate with:"
echo "  forge doctor"
echo "  forge backends"
