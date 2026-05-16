#!/usr/bin/env bash
set -euo pipefail

sudo apt update
sudo apt install -y \
  alsa-utils \
  build-essential \
  ca-certificates \
  curl \
  unzip

if ! command -v go >/dev/null 2>&1; then
  echo "Go is not installed. Install Go for linux/arm64, then rerun the assistant script." >&2
  echo "Recommended on Raspberry Pi OS: sudo apt install -y golang" >&2
  sudo apt install -y golang
fi

mkdir -p vosk-runtime models

if [[ ! -d vosk-runtime/vosk-linux-aarch64-0.3.45 ]]; then
  echo "Downloading Vosk Linux aarch64 runtime..."
  curl -L \
    -o /tmp/vosk-linux-aarch64-0.3.45.zip \
    https://github.com/alphacep/vosk-api/releases/download/v0.3.45/vosk-linux-aarch64-0.3.45.zip
  unzip -q /tmp/vosk-linux-aarch64-0.3.45.zip -d vosk-runtime
fi

if [[ ! -d models/vosk-model-small-en-us-0.15 ]]; then
  echo "Downloading Vosk small English model..."
  curl -L \
    -o /tmp/vosk-model-small-en-us-0.15.zip \
    https://alphacephei.com/vosk/models/vosk-model-small-en-us-0.15.zip
  unzip -q /tmp/vosk-model-small-en-us-0.15.zip -d models
fi

echo "Audio input devices:"
arecord -l || true

echo "Audio output devices:"
aplay -l || true

echo "Pi setup complete."
