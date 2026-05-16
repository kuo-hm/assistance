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

mkdir -p vosk-runtime models .gotmp/downloads

if [[ ! -d vosk-runtime/vosk-linux-aarch64-0.3.45 ]]; then
  echo "Downloading Vosk Linux aarch64 runtime..."
  rm -f .gotmp/downloads/vosk-linux-aarch64-0.3.45.zip
  curl -L \
    --fail \
    --retry 3 \
    -o .gotmp/downloads/vosk-linux-aarch64-0.3.45.zip \
    https://github.com/alphacep/vosk-api/releases/download/v0.3.45/vosk-linux-aarch64-0.3.45.zip
  unzip -q .gotmp/downloads/vosk-linux-aarch64-0.3.45.zip -d vosk-runtime
fi

if [[ ! -d models/vosk-model-small-en-us-0.15 ]]; then
  echo "Downloading Vosk small English model..."
  rm -f .gotmp/downloads/vosk-model-small-en-us-0.15.zip
  curl -L \
    --fail \
    --retry 3 \
    -o .gotmp/downloads/vosk-model-small-en-us-0.15.zip \
    https://alphacephei.com/vosk/models/vosk-model-small-en-us-0.15.zip
  rm -rf models/vosk-model-small-en-us-0.15
  unzip -q .gotmp/downloads/vosk-model-small-en-us-0.15.zip -d models
fi

echo "Audio input devices:"
arecord -l || true

echo "Audio output devices:"
aplay -l || true

echo "Pi setup complete."
