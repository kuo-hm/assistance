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
$env:ASSISTANT_LLM_PROVIDER="openai"
$env:OPENAI_API_KEY="your-key"
go run ./cmd/assistant
```

The app waits for Enter as the wake event, then asks you to type transcripts.

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
$env:ASSISTANT_LLM_PROVIDER="openai"
$env:OPENAI_API_KEY="your-openai-key"
$env:OPENAI_MODEL="gpt-4.1-mini"

$env:ASSISTANT_LLM_PROVIDER="gemini"
$env:GEMINI_API_KEY="your-gemini-key"
$env:GEMINI_MODEL="gemini-2.5-flash-lite"
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
