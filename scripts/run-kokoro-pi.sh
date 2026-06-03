#!/usr/bin/env bash
set -euo pipefail

# Get repository root path
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

echo "=== Raspberry Pi Kokoro TTS Setup & Run ==="

# 1. Install system dependencies if missing
MISSING_PKGS=()
for pkg in python3 python3-pip python3-venv espeak-ng libsndfile1 alsa-utils golang; do
  if ! dpkg -s "$pkg" >/dev/null 2>&1; then
    MISSING_PKGS+=("$pkg")
  fi
done

if [ ${#MISSING_PKGS[@]} -gt 0 ]; then
  echo "Installing missing system packages: ${MISSING_PKGS[*]}..."
  sudo apt update
  sudo apt install -y "${MISSING_PKGS[@]}"
fi

# 2. Set up Python virtual environment
VENV_DIR=".venv-kokoro"
if [ ! -d "$VENV_DIR" ]; then
  echo "Creating Python virtual environment at $VENV_DIR..."
  python3 -m venv "$VENV_DIR"
fi

# Activate virtual environment to check/install python packages
# shellcheck disable=SC1091
source "$VENV_DIR/bin/activate"

echo "Installing/updating Python dependencies (kokoro-onnx, soundfile)..."
pip install --upgrade pip
pip install kokoro-onnx soundfile

# 3. Create models directory if not exists
mkdir -p models

# 4. Load .env settings if present
if [[ -f .env ]]; then
  echo "Loading environment variables from .env..."
  while IFS= read -r line || [[ -n "$line" ]]; do
    [[ "$line" =~ ^[[:space:]]*$ ]] && continue
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    [[ "$line" != *"="* ]] && continue
    name="${line%%=*}"
    value="${line#*=}"
    name="$(echo "$name" | xargs)"
    [[ "$name" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]] || continue
    export "$name=$value"
  done < .env
fi

# 5. Configure Kokoro TTS defaults for Go runtime
export ASSISTANT_TTS_PROVIDER="kokoro"
export ASSISTANT_KOKORO_PYTHON="$ROOT/$VENV_DIR/bin/python"
export ASSISTANT_KOKORO_SCRIPT="scripts/kokoro_tts.py"
export ASSISTANT_KOKORO_MODEL="models/kokoro-v1.0.onnx"
export ASSISTANT_KOKORO_VOICES="models/voices-v1.0.bin"

# Default play command for Raspberry Pi (aplay plays WAV files natively)
export ASSISTANT_PLAY_COMMAND="${ASSISTANT_PLAY_COMMAND:-aplay {input}}"

# Default settings if not already defined
export GEMINI_API_KEY="${GEMINI_API_KEY:-}"
export ASSISTANT_MODE="${ASSISTANT_MODE:-turn}"
export ASSISTANT_TTS_VOICE_NAME="${ASSISTANT_TTS_VOICE_NAME:-af_bella}"

if [[ -z "$GEMINI_API_KEY" ]]; then
  echo "WARNING: GEMINI_API_KEY is not set. Please add it to your .env file or export it before running."
fi

# 6. Build the assistant binary for Linux
echo "Building Go assistant binary..."
mkdir -p .gotmp
go build -o .gotmp/assistant-pi ./cmd/assistant

echo "=== Setup complete! Starting assistant ==="
exec .gotmp/assistant-pi
