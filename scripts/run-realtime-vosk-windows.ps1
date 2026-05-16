$ErrorActionPreference = "Stop"

$repo = Split-Path -Parent $PSScriptRoot
$gccPath = "C:\msys64\ucrt64\bin"
$voskRuntimePath = Join-Path $repo "vosk-runtime\vosk-win64-0.3.45"

if (!(Test-Path (Join-Path $gccPath "gcc.exe"))) {
  throw "gcc.exe not found at $gccPath. Install MSYS2 UCRT64 or update `$gccPath in this script."
}
if (!(Test-Path (Join-Path $voskRuntimePath "libvosk.dll"))) {
  throw "libvosk.dll not found at $voskRuntimePath. Check the Vosk runtime folder."
}

Get-Content (Join-Path $repo ".env") | Where-Object { $_ -match '^[^#].+=' } | ForEach-Object {
  $name, $value = $_ -split '=', 2
  Set-Item -Path "Env:$name" -Value $value
}

$env:CGO_ENABLED = "1"
$env:ASSISTANT_MODE = "realtime"
$env:ASSISTANT_WAKE_PROVIDER = "voice"
$env:ASSISTANT_WAKE_STT_PROVIDER = "vosk"
if ([string]::IsNullOrWhiteSpace($env:ASSISTANT_WAKE_PHRASE)) {
  $env:ASSISTANT_WAKE_PHRASE = "wakeup"
}
$env:ASSISTANT_WAKE_RECORD_COMMAND = 'ffmpeg -nostdin -hide_banner -loglevel error -y -f dshow -i audio="Réseau de microphones (Technologie Intel® Smart Sound pour microphones numériques)" -t {seconds} -ac 1 -ar 16000 -sample_fmt s16 {output}'

Push-Location $repo
try {
  $outDir = Join-Path $repo ".gotmp"
  New-Item -ItemType Directory -Force -Path $outDir | Out-Null
  $binary = Join-Path $outDir "assistant-vosk.exe"

  $env:PATH = "$gccPath;$env:PATH"
  go build -tags vosk -o $binary ./cmd/assistant

  $env:PATH = "$voskRuntimePath;$gccPath;$env:PATH"
  & $binary
} finally {
  Pop-Location
}
