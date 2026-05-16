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
VOSK_LIB_DIR="${VOSK_LIB_DIR:-}"
VOSK_INCLUDE_DIR="${VOSK_INCLUDE_DIR:-}"

if [[ -z "$VOSK_LIB_DIR" ]]; then
  if [[ -f "$VOSK_RUNTIME/libvosk.so" ]]; then
    VOSK_LIB_DIR="$VOSK_RUNTIME"
  else
    VOSK_LIB_DIR="$(find "$ROOT/vosk-runtime" -name libvosk.so -type f -printf '%h\n' 2>/dev/null | head -n 1 || true)"
  fi
fi

if [[ -z "$VOSK_INCLUDE_DIR" ]]; then
  if [[ -f "$VOSK_RUNTIME/vosk_api.h" ]]; then
    VOSK_INCLUDE_DIR="$VOSK_RUNTIME"
  elif [[ -f "$ROOT/vosk-api-0.3.50/src/vosk_api.h" ]]; then
    VOSK_INCLUDE_DIR="$ROOT/vosk-api-0.3.50/src"
  else
    VOSK_INCLUDE_DIR="$(find "$ROOT" -name vosk_api.h -type f -printf '%h\n' 2>/dev/null | head -n 1 || true)"
  fi
fi

if [[ -z "$VOSK_LIB_DIR" || ! -f "$VOSK_LIB_DIR/libvosk.so" ]]; then
  echo "libvosk.so not found under $ROOT/vosk-runtime" >&2
  echo "Run ./scripts/setup-pi.sh first." >&2
  exit 1
fi

if [[ -z "$VOSK_INCLUDE_DIR" || ! -f "$VOSK_INCLUDE_DIR/vosk_api.h" ]]; then
  echo "vosk_api.h not found under $ROOT" >&2
  echo "Run ./scripts/setup-pi.sh first, or set VOSK_INCLUDE_DIR manually." >&2
  exit 1
fi

# Ignore Windows CGO flags from .env on Linux and use detected Pi paths.
export CGO_CPPFLAGS="-I$VOSK_INCLUDE_DIR"
export CGO_LDFLAGS="-L$VOSK_LIB_DIR -lvosk"
export LD_LIBRARY_PATH="$VOSK_LIB_DIR:${LD_LIBRARY_PATH:-}"

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

mkdir -p .gotmp
echo "Using Vosk include: $VOSK_INCLUDE_DIR"
echo "Using Vosk library: $VOSK_LIB_DIR"
go build -tags vosk -o .gotmp/assistant-vosk-pi ./cmd/assistant
exec .gotmp/assistant-vosk-pi
