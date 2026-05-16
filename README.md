# Go AI Voice Assistant

Voice assistant prototype for Windows development and Raspberry Pi Zero 2 W deployment.

## Capabilities

- Local wake event abstraction with console and external command modes.
- Continuous conversation loop after wake.
- Google Cloud Speech-to-Text adapter for `en-US`, `fr-FR`, and `ar-MA`.
- Gemini chat adapter using the official `google.golang.org/genai` SDK.
- Google Cloud Text-to-Speech adapter with approximate Darija fallback.
- SQLite memory for conversation turns, summaries, and durable facts.

## Quick Start

Console-only development mode:

```powershell
$env:ASSISTANT_LLM_PROVIDER="gemini"
$env:GEMINI_API_KEY="your-gemini-key"
go run ./cmd/assistant
```

The app waits for Enter as the wake event, then asks you to type transcripts.

Voice wake-up is available with `ASSISTANT_WAKE_PROVIDER=voice`. It records short microphone clips, transcribes them with the configured wake STT provider, and starts the assistant when the transcript contains `ASSISTANT_WAKE_PHRASE`.

Windows Vosk voice wake:

```powershell
.\scripts\run-realtime-vosk-windows.ps1
```

Manual equivalent:

```powershell
$env:PATH="C:\msys64\ucrt64\bin;C:\Users\Harmony\code\pers\assistance\vosk-runtime\vosk-win64-0.3.45;$env:PATH"
$env:CGO_ENABLED="1"
$env:ASSISTANT_WAKE_PROVIDER="voice"
$env:ASSISTANT_WAKE_PHRASE="hello"
$env:ASSISTANT_WAKE_ALIASES="hello there"
$env:ASSISTANT_WAKE_STT_PROVIDER="vosk"
$env:ASSISTANT_WAKE_RECORD_SECONDS="2"
$env:ASSISTANT_WAKE_MIN_CONFIDENCE="0.50"
$env:ASSISTANT_WAKE_DEBUG="true"
$env:ASSISTANT_RECORD_COMMAND='ffmpeg -nostdin -hide_banner -loglevel error -y -f dshow -i audio="Réseau de microphones (Technologie Intel® Smart Sound pour microphones numériques)" -t {seconds} -ac 1 -ar 16000 -sample_fmt s16 {output}'
go run -tags vosk ./cmd/assistant
```

With wake debug enabled, each wake clip prints what STT heard:

```text
wake heard: "wake up" confidence=0.84 language="en-US"
```

If Vosk hears a shorter form, add it as an alias:

```powershell
$env:ASSISTANT_WAKE_PHRASE="wakeup"
$env:ASSISTANT_WAKE_ALIASES="wake up,up"
```

Cloud wake-up uses the same provider but with Google STT:

```powershell
$env:ASSISTANT_WAKE_PROVIDER="voice"
$env:ASSISTANT_WAKE_STT_PROVIDER="google"
$env:GOOGLE_APPLICATION_CREDENTIALS="C:\path\to\service-account.json"
go run ./cmd/assistant
```

Switch between typed text and microphone recording with:

```powershell
$env:ASSISTANT_RECORDING_ENABLED="false" # typed text
$env:ASSISTANT_RECORDING_ENABLED="true"  # microphone + STT provider
```

Use local Windows speech while still printing replies:

```powershell
$env:ASSISTANT_TTS_PROVIDER="windows"
```

Switch the LLM provider:

```powershell
$env:ASSISTANT_LLM_PROVIDER="gemini"
$env:GEMINI_API_KEY="your-gemini-key"
$env:GEMINI_MODEL="gemini-2.5-flash-lite"

$env:ASSISTANT_LLM_PROVIDER="openai"
$env:OPENAI_API_KEY="your-openai-key"
$env:OPENAI_MODEL="gpt-4.1-mini"
```

Or load the checked-in `.env` file in PowerShell:

```powershell
Get-Content .env | Where-Object { $_ -match '^[^#].+=' } | ForEach-Object {
  $name, $value = $_ -split '=', 2
  Set-Item -Path "Env:$name" -Value $value
}
go run ./cmd/assistant
```

Cloud speech mode needs Google ADC credentials:

```powershell
$env:GOOGLE_APPLICATION_CREDENTIALS="C:\path\to\service-account.json"
$env:ASSISTANT_STT_PROVIDER="google"
$env:ASSISTANT_TTS_PROVIDER="google"
$env:ASSISTANT_RECORD_COMMAND='arecord -f S16_LE -r 16000 -c 1 -d {seconds} {output}'
$env:ASSISTANT_PLAY_COMMAND='ffplay -autoexit -nodisp {input}'
go run ./cmd/assistant
```

On Raspberry Pi, install an audio recorder/player such as `alsa-utils` and configure the record/play commands for the actual devices.

Pi helper scripts:

```bash
chmod +x scripts/setup-pi.sh scripts/run-realtime-vosk-pi.sh
./scripts/setup-pi.sh
GEMINI_API_KEY="your-gemini-key" ./scripts/run-realtime-vosk-pi.sh
```

`setup-pi.sh` installs ALSA/build tools, downloads the Vosk Linux `aarch64` runtime, and downloads `vosk-model-small-en-us-0.15`.

If your ALSA device is not the default, override the commands:

```bash
export ASSISTANT_REALTIME_INPUT_COMMAND='arecord -D plughw:1,0 -f S16_LE -r {sample_rate} -c 1 -t raw'
export ASSISTANT_RECORD_COMMAND='arecord -D plughw:1,0 -f S16_LE -r 16000 -c 1 -d {seconds} {output}'
export ASSISTANT_REALTIME_OUTPUT_COMMAND='aplay -D plughw:0,0 -f S16_LE -r {sample_rate} -c 1'
./scripts/run-realtime-vosk-pi.sh
```

## Realtime Speech-to-Speech

Realtime mode keeps the wake behavior, then opens a Gemini Live audio session. It streams raw microphone PCM to Gemini and plays Gemini's native audio response while still printing transcripts when Gemini sends them.

Raspberry Pi setup:

```bash
sudo apt update
sudo apt install -y golang alsa-utils
export ASSISTANT_MODE=realtime
export ASSISTANT_REALTIME_PROVIDER=gemini
export GEMINI_API_KEY="your-gemini-key"
export GEMINI_LIVE_MODEL="gemini-2.5-flash-native-audio-preview-12-2025"
export GEMINI_LIVE_VOICE="Kore"
export ASSISTANT_REALTIME_INPUT_COMMAND='arecord -f S16_LE -r {sample_rate} -c 1 -t raw'
export ASSISTANT_REALTIME_OUTPUT_COMMAND='aplay -f S16_LE -r {sample_rate} -c 1'
go run ./cmd/assistant
```

Windows realtime test with FFmpeg/FFplay:

```powershell
$env:ASSISTANT_MODE="realtime"
$env:ASSISTANT_REALTIME_PROVIDER="gemini"
$env:GEMINI_API_KEY="your-gemini-key"
$env:ASSISTANT_REALTIME_INPUT_COMMAND='ffmpeg -hide_banner -loglevel warning -f dshow -i audio="Réseau de microphones (Technologie Intel® Smart Sound pour microphones numériques)" -ac 1 -ar {sample_rate} -f s16le -'
$env:ASSISTANT_REALTIME_OUTPUT_COMMAND='ffplay -hide_banner -loglevel error -autoexit -nodisp -f s16le -ar {sample_rate} -ch_layout mono -i -'
go run ./cmd/assistant
```

Use `ASSISTANT_MODE=turn` to return to the existing text/mic pipeline.

## Offline STT With Vosk

Vosk is optional because it needs the native `libvosk` runtime. Download a model from `https://alphacephei.com/vosk/models`, extract it under `models/`, then set:

```powershell
$env:ASSISTANT_STT_PROVIDER="vosk"
$env:ASSISTANT_VOSK_MODEL_PATH="models/vosk-model-small-en-us-0.15"
$env:ASSISTANT_RECORD_COMMAND='arecord -f S16_LE -r 16000 -c 1 -d {seconds} {output}'
go run -tags vosk ./cmd/assistant
```

On Windows, put `libvosk.dll` somewhere in `PATH` before running with `-tags vosk`.
Vosk's Go binding also requires CGO and a Windows C compiler such as MinGW-w64/MSYS2. Set:

```powershell
$env:CGO_ENABLED="1"
$env:CGO_CPPFLAGS="-IC:\Users\Harmony\code\pers\assistance\vosk-runtime\vosk-win64-0.3.45"
$env:CGO_LDFLAGS="-LC:\Users\Harmony\code\pers\assistance\vosk-runtime\vosk-win64-0.3.45 -lvosk"
```

## Hybrid STT

Hybrid mode tries Vosk locally for English/French first, then falls back to Google STT for Moroccan Arabic when the local transcript is empty or below the confidence threshold.

```powershell
$env:ASSISTANT_STT_PROVIDER="hybrid"
$env:ASSISTANT_HYBRID_LOCAL_LANGUAGES="en-US,fr-FR"
$env:ASSISTANT_HYBRID_CLOUD_LANGUAGES="ar-MA"
$env:ASSISTANT_HYBRID_MIN_CONFIDENCE="0.65"
$env:ASSISTANT_VOSK_MODEL_PATH="models/vosk-model-small-en-us-0.15"
$env:GOOGLE_APPLICATION_CREDENTIALS="C:\path\to\service-account.json"
go run -tags vosk ./cmd/assistant
```
