package config

import (
	"os"
	"testing"
)

func TestLoadUsesDefaultsAndEnv(t *testing.T) {
	clearAssistantEnv(t)
	t.Setenv("GEMINI_API_KEY", "test-key")
	t.Setenv("ASSISTANT_LANGUAGES", "en-US,fr-FR,ar-MA")
	t.Setenv("ASSISTANT_SQLITE_PATH", "test.db")
	t.Setenv("ASSISTANT_CONFIG", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.GeminiAPIKey != "test-key" {
		t.Fatalf("GeminiAPIKey = %q", cfg.GeminiAPIKey)
	}
	if cfg.WakePhrase != "hello there" {
		t.Fatalf("WakePhrase = %q", cfg.WakePhrase)
	}
	if cfg.Mode != "turn" {
		t.Fatalf("Mode = %q", cfg.Mode)
	}
	if cfg.RealtimeProvider != "gemini" {
		t.Fatalf("RealtimeProvider = %q", cfg.RealtimeProvider)
	}
	if len(cfg.Languages) != 3 {
		t.Fatalf("Languages length = %d", len(cfg.Languages))
	}
	if cfg.KokoroPython != "python" {
		t.Fatalf("KokoroPython = %q", cfg.KokoroPython)
	}
	if cfg.KokoroScript != "scripts/kokoro_tts.py" {
		t.Fatalf("KokoroScript = %q", cfg.KokoroScript)
	}
	if cfg.KokoroModel != "models/kokoro-v1.0.onnx" {
		t.Fatalf("KokoroModel = %q", cfg.KokoroModel)
	}
	if cfg.KokoroVoices != "models/voices-v1.0.bin" {
		t.Fatalf("KokoroVoices = %q", cfg.KokoroVoices)
	}
}

func TestLoadConfigFile(t *testing.T) {
	clearAssistantEnv(t)

	file, err := os.CreateTemp(t.TempDir(), "config-*.json")
	if err != nil {
		t.Fatal(err)
	}
	_, err = file.WriteString(`{
		"gemini_api_key": "file-key",
		"sqlite_path": "memory.db",
		"silence_timeout": "2s",
		"languages": ["fr-FR"]
	}`)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ASSISTANT_CONFIG", file.Name())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.GeminiAPIKey != "file-key" {
		t.Fatalf("GeminiAPIKey = %q", cfg.GeminiAPIKey)
	}
	if cfg.Languages[0] != "fr-FR" {
		t.Fatalf("Languages[0] = %q", cfg.Languages[0])
	}
}

func TestLoadKokoroConfig(t *testing.T) {
	clearAssistantEnv(t)
	t.Setenv("GEMINI_API_KEY", "test-key")
	t.Setenv("ASSISTANT_KOKORO_PYTHON", "python3")
	t.Setenv("ASSISTANT_KOKORO_SCRIPT", "custom_kokoro.py")
	t.Setenv("ASSISTANT_KOKORO_MODEL", "custom_model.onnx")
	t.Setenv("ASSISTANT_KOKORO_VOICES", "custom_voices.bin")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.KokoroPython != "python3" {
		t.Fatalf("KokoroPython = %q", cfg.KokoroPython)
	}
	if cfg.KokoroScript != "custom_kokoro.py" {
		t.Fatalf("KokoroScript = %q", cfg.KokoroScript)
	}
	if cfg.KokoroModel != "custom_model.onnx" {
		t.Fatalf("KokoroModel = %q", cfg.KokoroModel)
	}
	if cfg.KokoroVoices != "custom_voices.bin" {
		t.Fatalf("KokoroVoices = %q", cfg.KokoroVoices)
	}
}

func clearAssistantEnv(t *testing.T) {
	t.Helper()
	names := []string{
		"GEMINI_API_KEY",
		"GEMINI_MODEL",
		"ASSISTANT_LLM_PROVIDER",
		"OPENAI_API_KEY",
		"OPENAI_MODEL",
		"GOOGLE_APPLICATION_CREDENTIALS",
		"ASSISTANT_CONFIG",
		"ASSISTANT_MODE",
		"ASSISTANT_WAKE_PROVIDER",
		"ASSISTANT_WAKE_PHRASE",
		"ASSISTANT_WAKE_ALIASES",
		"ASSISTANT_WAKE_COMMAND",
		"ASSISTANT_WAKE_STT_PROVIDER",
		"ASSISTANT_WAKE_RECORD_COMMAND",
		"ASSISTANT_WAKE_RECORD_SECONDS",
		"ASSISTANT_WAKE_MIN_CONFIDENCE",
		"ASSISTANT_WAKE_DEBUG",
		"ASSISTANT_STT_PROVIDER",
		"ASSISTANT_TTS_PROVIDER",
		"ASSISTANT_SQLITE_PATH",
		"ASSISTANT_RECORDING_ENABLED",
		"ASSISTANT_LANGUAGES",
		"ASSISTANT_VOSK_MODEL_PATH",
		"ASSISTANT_VOSK_SAMPLE_RATE",
		"ASSISTANT_HYBRID_LOCAL_LANGUAGES",
		"ASSISTANT_HYBRID_CLOUD_LANGUAGES",
		"ASSISTANT_HYBRID_MIN_CONFIDENCE",
		"ASSISTANT_SILENCE_TIMEOUT",
		"ASSISTANT_SESSION_IDLE_TIMEOUT",
		"ASSISTANT_MAX_TURNS",
		"ASSISTANT_RECORD_COMMAND",
		"ASSISTANT_RECORD_SECONDS",
		"ASSISTANT_PLAY_COMMAND",
		"ASSISTANT_TTS_LANGUAGE_CODE",
		"ASSISTANT_TTS_VOICE_NAME",
		"ASSISTANT_REALTIME_PROVIDER",
		"GEMINI_LIVE_MODEL",
		"GEMINI_LIVE_VOICE",
		"ASSISTANT_INPUT_SAMPLE_RATE",
		"ASSISTANT_OUTPUT_SAMPLE_RATE",
		"ASSISTANT_REALTIME_CHUNK_BYTES",
		"ASSISTANT_REALTIME_INPUT_COMMAND",
		"ASSISTANT_REALTIME_OUTPUT_COMMAND",
		"ASSISTANT_KOKORO_PYTHON",
		"ASSISTANT_KOKORO_SCRIPT",
		"ASSISTANT_KOKORO_MODEL",
		"ASSISTANT_KOKORO_VOICES",
	}
	for _, name := range names {
		t.Setenv(name, "")
	}
}
