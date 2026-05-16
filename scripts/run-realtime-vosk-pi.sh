#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

if [[ -f .env ]]; then
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

export CGO_ENABLED="${CGO_ENABLED:-1}"
export ASSISTANT_MODE="${ASSISTANT_MODE:-realtime}"
export ASSISTANT_WAKE_PROVIDER="${ASSISTANT_WAKE_PROVIDER:-voice}"
export ASSISTANT_WAKE_STT_PROVIDER="${ASSISTANT_WAKE_STT_PROVIDER:-vosk}"
export ASSISTANT_WAKE_PHRASE="${ASSISTANT_WAKE_PHRASE:-wakeup}"
export ASSISTANT_WAKE_ALIASES="${ASSISTANT_WAKE_ALIASES:-wake up,up}"
export ASSISTANT_WAKE_RECORD_SECONDS="${ASSISTANT_WAKE_RECORD_SECONDS:-3}"
export ASSISTANT_WAKE_DEBUG="${ASSISTANT_WAKE_DEBUG:-true}"
export ASSISTANT_RECORD_SECONDS="${ASSISTANT_RECORD_SECONDS:-12}"
export ASSISTANT_INPUT_SAMPLE_RATE="${ASSISTANT_INPUT_SAMPLE_RATE:-16000}"
export ASSISTANT_OUTPUT_SAMPLE_RATE="${ASSISTANT_OUTPUT_SAMPLE_RATE:-24000}"
export ASSISTANT_REALTIME_CHUNK_BYTES="${ASSISTANT_REALTIME_CHUNK_BYTES:-3200}"
export ASSISTANT_REALTIME_PROVIDER="${ASSISTANT_REALTIME_PROVIDER:-gemini}"
export ASSISTANT_REALTIME_INPUT_COMMAND="${ASSISTANT_REALTIME_INPUT_COMMAND:-arecord -f S16_LE -r {sample_rate} -c 1 -t raw}"
export ASSISTANT_REALTIME_OUTPUT_COMMAND="${ASSISTANT_REALTIME_OUTPUT_COMMAND:-aplay -f S16_LE -r {sample_rate} -c 1}"
export ASSISTANT_RECORD_COMMAND="${ASSISTANT_RECORD_COMMAND:-arecord -f S16_LE -r 16000 -c 1 -d {seconds} {output}}"
export ASSISTANT_WAKE_RECORD_COMMAND="${ASSISTANT_WAKE_RECORD_COMMAND:-$ASSISTANT_RECORD_COMMAND}"
export ASSISTANT_VOSK_MODEL_PATH="${ASSISTANT_VOSK_MODEL_PATH:-models/vosk-model-small-en-us-0.15}"
export ASSISTANT_VOSK_SAMPLE_RATE="${ASSISTANT_VOSK_SAMPLE_RATE:-16000}"

VOSK_RUNTIME="${VOSK_RUNTIME:-$ROOT/vosk-runtime/vosk-linux-aarch64-0.3.45}"
export CGO_CPPFLAGS="${CGO_CPPFLAGS:--I$VOSK_RUNTIME}"
export CGO_LDFLAGS="${CGO_LDFLAGS:--L$VOSK_RUNTIME -lvosk}"
export LD_LIBRARY_PATH="$VOSK_RUNTIME:${LD_LIBRARY_PATH:-}"

if [[ -z "${GEMINI_API_KEY:-}" ]]; then
  echo "GEMINI_API_KEY is required. Put it in .env or export it before running." >&2
  exit 1
fi

if ! command -v arecord >/dev/null 2>&1; then
  echo "arecord not found. Install ALSA tools: sudo apt install -y alsa-utils" >&2
  exit 1
fi

if ! command -v aplay >/dev/null 2>&1; then
  echo "aplay not found. Install ALSA tools: sudo apt install -y alsa-utils" >&2
  exit 1
fi

if [[ ! -d "$ASSISTANT_VOSK_MODEL_PATH" ]]; then
  echo "Vosk model not found: $ASSISTANT_VOSK_MODEL_PATH" >&2
  echo "Download/extract a Linux-compatible model under models/ first." >&2
  exit 1
fi

if [[ ! -f "$VOSK_RUNTIME/libvosk.so" || ! -f "$VOSK_RUNTIME/vosk_api.h" ]]; then
  echo "Vosk runtime not found at: $VOSK_RUNTIME" >&2
  echo "Run ./scripts/setup-pi.sh first." >&2
  exit 1
fi

mkdir -p .gotmp
go build -tags vosk -o .gotmp/assistant-vosk-pi ./cmd/assistant
exec .gotmp/assistant-vosk-pi
