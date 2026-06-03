package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config contains runtime settings for the assistant.
type Config struct {
	GeminiAPIKey          string        `json:"gemini_api_key"`
	GeminiModel           string        `json:"gemini_model"`
	LLMProvider           string        `json:"llm_provider"`
	OpenAIAPIKey          string        `json:"openai_api_key"`
	OpenAIModel           string        `json:"openai_model"`
	GoogleCredentialsFile string        `json:"google_credentials_file"`
	Mode                  string        `json:"mode"`
	WakeProvider          string        `json:"wake_provider"`
	WakePhrase            string        `json:"wake_phrase"`
	WakeAliases           []string      `json:"wake_aliases"`
	WakeCommand           string        `json:"wake_command"`
	WakeSTTProvider       string        `json:"wake_stt_provider"`
	WakeRecordCommand     string        `json:"wake_record_command"`
	WakeRecordSeconds     int           `json:"wake_record_seconds"`
	WakeMinConfidence     float32       `json:"wake_min_confidence"`
	WakeDebug             bool          `json:"wake_debug"`
	STTProvider           string        `json:"stt_provider"`
	TTSProvider           string        `json:"tts_provider"`
	SQLitePath            string        `json:"sqlite_path"`
	RecordingEnabled      bool          `json:"recording_enabled"`
	Languages             []string      `json:"languages"`
	VoskModelPath         string        `json:"vosk_model_path"`
	VoskSampleRate        float64       `json:"vosk_sample_rate"`
	HybridLocalLanguages  []string      `json:"hybrid_local_languages"`
	HybridCloudLanguages  []string      `json:"hybrid_cloud_languages"`
	HybridMinConfidence   float32       `json:"hybrid_min_confidence"`
	SilenceTimeout        time.Duration `json:"silence_timeout"`
	SessionIdleTimeout    time.Duration `json:"session_idle_timeout"`
	MaxTurnsPerSession    int           `json:"max_turns_per_session"`
	RecordCommand         string        `json:"record_command"`
	RecordSeconds         int           `json:"record_seconds"`
	PlayCommand           string        `json:"play_command"`
	TTSLanguageCode       string        `json:"tts_language_code"`
	TTSVoiceName          string        `json:"tts_voice_name"`
	RealtimeProvider      string        `json:"realtime_provider"`
	GeminiLiveModel       string        `json:"gemini_live_model"`
	GeminiLiveVoice       string        `json:"gemini_live_voice"`
	InputSampleRate       int           `json:"input_sample_rate"`
	OutputSampleRate      int           `json:"output_sample_rate"`
	RealtimeInputCommand  string        `json:"realtime_input_command"`
	RealtimeOutputCommand string        `json:"realtime_output_command"`
	RealtimeChunkBytes    int           `json:"realtime_chunk_bytes"`
	KokoroPython          string        `json:"kokoro_python"`
	KokoroScript          string        `json:"kokoro_script"`
	KokoroModel           string        `json:"kokoro_model"`
	KokoroVoices          string        `json:"kokoro_voices"`
}

type fileConfig struct {
	GeminiAPIKey          string   `json:"gemini_api_key"`
	GeminiModel           string   `json:"gemini_model"`
	LLMProvider           string   `json:"llm_provider"`
	OpenAIAPIKey          string   `json:"openai_api_key"`
	OpenAIModel           string   `json:"openai_model"`
	GoogleCredentialsFile string   `json:"google_credentials_file"`
	Mode                  string   `json:"mode"`
	WakeProvider          string   `json:"wake_provider"`
	WakePhrase            string   `json:"wake_phrase"`
	WakeAliases           []string `json:"wake_aliases"`
	WakeCommand           string   `json:"wake_command"`
	WakeSTTProvider       string   `json:"wake_stt_provider"`
	WakeRecordCommand     string   `json:"wake_record_command"`
	WakeRecordSeconds     int      `json:"wake_record_seconds"`
	WakeMinConfidence     float32  `json:"wake_min_confidence"`
	WakeDebug             *bool    `json:"wake_debug"`
	STTProvider           string   `json:"stt_provider"`
	TTSProvider           string   `json:"tts_provider"`
	SQLitePath            string   `json:"sqlite_path"`
	RecordingEnabled      *bool    `json:"recording_enabled"`
	Languages             []string `json:"languages"`
	VoskModelPath         string   `json:"vosk_model_path"`
	VoskSampleRate        float64  `json:"vosk_sample_rate"`
	HybridLocalLanguages  []string `json:"hybrid_local_languages"`
	HybridCloudLanguages  []string `json:"hybrid_cloud_languages"`
	HybridMinConfidence   float32  `json:"hybrid_min_confidence"`
	SilenceTimeout        string   `json:"silence_timeout"`
	SessionIdleTimeout    string   `json:"session_idle_timeout"`
	MaxTurnsPerSession    int      `json:"max_turns_per_session"`
	RecordCommand         string   `json:"record_command"`
	RecordSeconds         int      `json:"record_seconds"`
	PlayCommand           string   `json:"play_command"`
	TTSLanguageCode       string   `json:"tts_language_code"`
	TTSVoiceName          string   `json:"tts_voice_name"`
	RealtimeProvider      string   `json:"realtime_provider"`
	GeminiLiveModel       string   `json:"gemini_live_model"`
	GeminiLiveVoice       string   `json:"gemini_live_voice"`
	InputSampleRate       int      `json:"input_sample_rate"`
	OutputSampleRate      int      `json:"output_sample_rate"`
	RealtimeInputCommand  string   `json:"realtime_input_command"`
	RealtimeOutputCommand string   `json:"realtime_output_command"`
	RealtimeChunkBytes    int      `json:"realtime_chunk_bytes"`
	KokoroPython          string   `json:"kokoro_python"`
	KokoroScript          string   `json:"kokoro_script"`
	KokoroModel           string   `json:"kokoro_model"`
	KokoroVoices          string   `json:"kokoro_voices"`
}

// Load reads defaults, an optional JSON config file, and environment overrides.
func Load() (Config, error) {
	cfg := defaults()

	if path := os.Getenv("ASSISTANT_CONFIG"); path != "" {
		if err := loadFile(path, &cfg); err != nil {
			return Config{}, err
		}
	}

	applyEnv(&cfg)

	if cfg.GeminiAPIKey == "" {
		return Config{}, errors.New("GEMINI_API_KEY is required")
	}
	if len(cfg.Languages) == 0 {
		return Config{}, errors.New("at least one STT language is required")
	}
	if cfg.SQLitePath == "" {
		return Config{}, errors.New("sqlite path is required")
	}
	if cfg.RecordSeconds <= 0 {
		return Config{}, errors.New("record seconds must be positive")
	}
	if cfg.MaxTurnsPerSession <= 0 {
		return Config{}, errors.New("max turns per session must be positive")
	}
	if cfg.WakeRecordSeconds <= 0 {
		return Config{}, errors.New("wake record seconds must be positive")
	}
	if cfg.WakeProvider == "voice" && cfg.WakeRecordCommand == "" && cfg.RecordCommand == "" {
		return Config{}, errors.New("ASSISTANT_WAKE_RECORD_COMMAND or ASSISTANT_RECORD_COMMAND is required when ASSISTANT_WAKE_PROVIDER=voice")
	}
	if cfg.InputSampleRate <= 0 {
		return Config{}, errors.New("input sample rate must be positive")
	}
	if cfg.OutputSampleRate <= 0 {
		return Config{}, errors.New("output sample rate must be positive")
	}
	if cfg.RealtimeChunkBytes <= 0 {
		return Config{}, errors.New("realtime chunk bytes must be positive")
	}

	return cfg, nil
}

func defaults() Config {
	return Config{
		GeminiModel:       "gemini-2.5-flash",
		LLMProvider:       "gemini",
		OpenAIModel:       "gpt-4.1-mini",
		Mode:              "turn",
		WakeProvider:      "console",
		WakePhrase:        "hello there",
		WakeRecordSeconds: 2,
		STTProvider:       "console",
		TTSProvider:       "console",
		SQLitePath:        "assistant.db",
		RecordingEnabled:  false,
		Languages:         []string{"en-US", "fr-FR", "ar-MA"},
		VoskSampleRate:    16000,
		HybridLocalLanguages: []string{
			"en-US",
			"fr-FR",
		},
		HybridCloudLanguages:  []string{"ar-MA"},
		HybridMinConfidence:   0.65,
		SilenceTimeout:        3 * time.Second,
		SessionIdleTimeout:    45 * time.Second,
		MaxTurnsPerSession:    12,
		RecordSeconds:         12,
		TTSLanguageCode:       "ar-XA",
		TTSVoiceName:          "",
		RealtimeProvider:      "gemini",
		GeminiLiveModel:       "gemini-2.5-flash-native-audio-preview-12-2025",
		GeminiLiveVoice:       "Kore",
		InputSampleRate:       16000,
		OutputSampleRate:      24000,
		RealtimeInputCommand:  "arecord -f S16_LE -r {sample_rate} -c 1 -t raw",
		RealtimeOutputCommand: "aplay -f S16_LE -r {sample_rate} -c 1",
		RealtimeChunkBytes:    3200,
		KokoroPython:          "python",
		KokoroScript:          "scripts/kokoro_tts.py",
		KokoroModel:           "models/kokoro-v1.0.onnx",
		KokoroVoices:          "models/voices-v1.0.bin",
	}
}

func loadFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config file: %w", err)
	}

	var fc fileConfig
	if err := json.Unmarshal(data, &fc); err != nil {
		return fmt.Errorf("parse config file: %w", err)
	}

	if fc.GeminiAPIKey != "" {
		cfg.GeminiAPIKey = fc.GeminiAPIKey
	}
	if fc.GeminiModel != "" {
		cfg.GeminiModel = fc.GeminiModel
	}
	if fc.LLMProvider != "" {
		cfg.LLMProvider = fc.LLMProvider
	}
	if fc.OpenAIAPIKey != "" {
		cfg.OpenAIAPIKey = fc.OpenAIAPIKey
	}
	if fc.OpenAIModel != "" {
		cfg.OpenAIModel = fc.OpenAIModel
	}
	if fc.GoogleCredentialsFile != "" {
		cfg.GoogleCredentialsFile = fc.GoogleCredentialsFile
	}
	if fc.Mode != "" {
		cfg.Mode = fc.Mode
	}
	if fc.WakeProvider != "" {
		cfg.WakeProvider = fc.WakeProvider
	}
	if fc.WakePhrase != "" {
		cfg.WakePhrase = fc.WakePhrase
	}
	if len(fc.WakeAliases) > 0 {
		cfg.WakeAliases = fc.WakeAliases
	}
	if fc.WakeCommand != "" {
		cfg.WakeCommand = fc.WakeCommand
	}
	if fc.WakeSTTProvider != "" {
		cfg.WakeSTTProvider = fc.WakeSTTProvider
	}
	if fc.WakeRecordCommand != "" {
		cfg.WakeRecordCommand = fc.WakeRecordCommand
	}
	if fc.WakeRecordSeconds > 0 {
		cfg.WakeRecordSeconds = fc.WakeRecordSeconds
	}
	if fc.WakeMinConfidence > 0 {
		cfg.WakeMinConfidence = fc.WakeMinConfidence
	}
	if fc.WakeDebug != nil {
		cfg.WakeDebug = *fc.WakeDebug
	}
	if fc.STTProvider != "" {
		cfg.STTProvider = fc.STTProvider
	}
	if fc.TTSProvider != "" {
		cfg.TTSProvider = fc.TTSProvider
	}
	if fc.SQLitePath != "" {
		cfg.SQLitePath = fc.SQLitePath
	}
	if fc.RecordingEnabled != nil {
		cfg.RecordingEnabled = *fc.RecordingEnabled
	}
	if len(fc.Languages) > 0 {
		cfg.Languages = fc.Languages
	}
	if fc.VoskModelPath != "" {
		cfg.VoskModelPath = fc.VoskModelPath
	}
	if fc.VoskSampleRate > 0 {
		cfg.VoskSampleRate = fc.VoskSampleRate
	}
	if len(fc.HybridLocalLanguages) > 0 {
		cfg.HybridLocalLanguages = fc.HybridLocalLanguages
	}
	if len(fc.HybridCloudLanguages) > 0 {
		cfg.HybridCloudLanguages = fc.HybridCloudLanguages
	}
	if fc.HybridMinConfidence > 0 {
		cfg.HybridMinConfidence = fc.HybridMinConfidence
	}
	if fc.SilenceTimeout != "" {
		duration, err := time.ParseDuration(fc.SilenceTimeout)
		if err != nil {
			return fmt.Errorf("parse silence_timeout: %w", err)
		}
		cfg.SilenceTimeout = duration
	}
	if fc.SessionIdleTimeout != "" {
		duration, err := time.ParseDuration(fc.SessionIdleTimeout)
		if err != nil {
			return fmt.Errorf("parse session_idle_timeout: %w", err)
		}
		cfg.SessionIdleTimeout = duration
	}
	if fc.MaxTurnsPerSession > 0 {
		cfg.MaxTurnsPerSession = fc.MaxTurnsPerSession
	}
	if fc.RecordCommand != "" {
		cfg.RecordCommand = fc.RecordCommand
	}
	if fc.RecordSeconds > 0 {
		cfg.RecordSeconds = fc.RecordSeconds
	}
	if fc.PlayCommand != "" {
		cfg.PlayCommand = fc.PlayCommand
	}
	if fc.TTSLanguageCode != "" {
		cfg.TTSLanguageCode = fc.TTSLanguageCode
	}
	if fc.TTSVoiceName != "" {
		cfg.TTSVoiceName = fc.TTSVoiceName
	}
	if fc.RealtimeProvider != "" {
		cfg.RealtimeProvider = fc.RealtimeProvider
	}
	if fc.GeminiLiveModel != "" {
		cfg.GeminiLiveModel = fc.GeminiLiveModel
	}
	if fc.GeminiLiveVoice != "" {
		cfg.GeminiLiveVoice = fc.GeminiLiveVoice
	}
	if fc.InputSampleRate > 0 {
		cfg.InputSampleRate = fc.InputSampleRate
	}
	if fc.OutputSampleRate > 0 {
		cfg.OutputSampleRate = fc.OutputSampleRate
	}
	if fc.RealtimeInputCommand != "" {
		cfg.RealtimeInputCommand = fc.RealtimeInputCommand
	}
	if fc.RealtimeOutputCommand != "" {
		cfg.RealtimeOutputCommand = fc.RealtimeOutputCommand
	}
	if fc.RealtimeChunkBytes > 0 {
		cfg.RealtimeChunkBytes = fc.RealtimeChunkBytes
	}
	if fc.KokoroPython != "" {
		cfg.KokoroPython = fc.KokoroPython
	}
	if fc.KokoroScript != "" {
		cfg.KokoroScript = fc.KokoroScript
	}
	if fc.KokoroModel != "" {
		cfg.KokoroModel = fc.KokoroModel
	}
	if fc.KokoroVoices != "" {
		cfg.KokoroVoices = fc.KokoroVoices
	}
	return nil
}

func applyEnv(cfg *Config) {
	envString("GEMINI_API_KEY", &cfg.GeminiAPIKey)
	envString("GEMINI_MODEL", &cfg.GeminiModel)
	envString("ASSISTANT_LLM_PROVIDER", &cfg.LLMProvider)
	envString("OPENAI_API_KEY", &cfg.OpenAIAPIKey)
	envString("OPENAI_MODEL", &cfg.OpenAIModel)
	envString("GOOGLE_APPLICATION_CREDENTIALS", &cfg.GoogleCredentialsFile)
	envString("ASSISTANT_MODE", &cfg.Mode)
	envString("ASSISTANT_WAKE_PROVIDER", &cfg.WakeProvider)
	envString("ASSISTANT_WAKE_PHRASE", &cfg.WakePhrase)
	envString("ASSISTANT_WAKE_COMMAND", &cfg.WakeCommand)
	envString("ASSISTANT_WAKE_STT_PROVIDER", &cfg.WakeSTTProvider)
	envString("ASSISTANT_WAKE_RECORD_COMMAND", &cfg.WakeRecordCommand)
	envString("ASSISTANT_STT_PROVIDER", &cfg.STTProvider)
	envString("ASSISTANT_TTS_PROVIDER", &cfg.TTSProvider)
	envString("ASSISTANT_SQLITE_PATH", &cfg.SQLitePath)
	envString("ASSISTANT_RECORD_COMMAND", &cfg.RecordCommand)
	envString("ASSISTANT_PLAY_COMMAND", &cfg.PlayCommand)
	envString("ASSISTANT_TTS_LANGUAGE_CODE", &cfg.TTSLanguageCode)
	envString("ASSISTANT_TTS_VOICE_NAME", &cfg.TTSVoiceName)
	envString("ASSISTANT_VOSK_MODEL_PATH", &cfg.VoskModelPath)
	envString("ASSISTANT_REALTIME_PROVIDER", &cfg.RealtimeProvider)
	envString("GEMINI_LIVE_MODEL", &cfg.GeminiLiveModel)
	envString("GEMINI_LIVE_VOICE", &cfg.GeminiLiveVoice)
	envString("ASSISTANT_REALTIME_INPUT_COMMAND", &cfg.RealtimeInputCommand)
	envString("ASSISTANT_REALTIME_OUTPUT_COMMAND", &cfg.RealtimeOutputCommand)
	envString("ASSISTANT_KOKORO_PYTHON", &cfg.KokoroPython)
	envString("ASSISTANT_KOKORO_SCRIPT", &cfg.KokoroScript)
	envString("ASSISTANT_KOKORO_MODEL", &cfg.KokoroModel)
	envString("ASSISTANT_KOKORO_VOICES", &cfg.KokoroVoices)

	if raw := os.Getenv("ASSISTANT_RECORDING_ENABLED"); raw != "" {
		if value, err := strconv.ParseBool(raw); err == nil {
			cfg.RecordingEnabled = value
		}
	}
	if raw := os.Getenv("ASSISTANT_WAKE_DEBUG"); raw != "" {
		if value, err := strconv.ParseBool(raw); err == nil {
			cfg.WakeDebug = value
		}
	}
	if raw := os.Getenv("ASSISTANT_LANGUAGES"); raw != "" {
		cfg.Languages = splitCSV(raw)
	}
	if raw := os.Getenv("ASSISTANT_WAKE_ALIASES"); raw != "" {
		cfg.WakeAliases = splitCSV(raw)
	}
	if raw := os.Getenv("ASSISTANT_HYBRID_LOCAL_LANGUAGES"); raw != "" {
		cfg.HybridLocalLanguages = splitCSV(raw)
	}
	if raw := os.Getenv("ASSISTANT_HYBRID_CLOUD_LANGUAGES"); raw != "" {
		cfg.HybridCloudLanguages = splitCSV(raw)
	}
	if raw := os.Getenv("ASSISTANT_SILENCE_TIMEOUT"); raw != "" {
		if duration, err := time.ParseDuration(raw); err == nil {
			cfg.SilenceTimeout = duration
		}
	}
	if raw := os.Getenv("ASSISTANT_SESSION_IDLE_TIMEOUT"); raw != "" {
		if duration, err := time.ParseDuration(raw); err == nil {
			cfg.SessionIdleTimeout = duration
		}
	}
	if raw := os.Getenv("ASSISTANT_MAX_TURNS"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			cfg.MaxTurnsPerSession = value
		}
	}
	if raw := os.Getenv("ASSISTANT_RECORD_SECONDS"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			cfg.RecordSeconds = value
		}
	}
	if raw := os.Getenv("ASSISTANT_WAKE_RECORD_SECONDS"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			cfg.WakeRecordSeconds = value
		}
	}
	if raw := os.Getenv("ASSISTANT_INPUT_SAMPLE_RATE"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			cfg.InputSampleRate = value
		}
	}
	if raw := os.Getenv("ASSISTANT_OUTPUT_SAMPLE_RATE"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			cfg.OutputSampleRate = value
		}
	}
	if raw := os.Getenv("ASSISTANT_REALTIME_CHUNK_BYTES"); raw != "" {
		if value, err := strconv.Atoi(raw); err == nil {
			cfg.RealtimeChunkBytes = value
		}
	}
	if raw := os.Getenv("ASSISTANT_VOSK_SAMPLE_RATE"); raw != "" {
		if value, err := strconv.ParseFloat(raw, 64); err == nil {
			cfg.VoskSampleRate = value
		}
	}
	if raw := os.Getenv("ASSISTANT_HYBRID_MIN_CONFIDENCE"); raw != "" {
		if value, err := strconv.ParseFloat(raw, 32); err == nil {
			cfg.HybridMinConfidence = float32(value)
		}
	}
	if raw := os.Getenv("ASSISTANT_WAKE_MIN_CONFIDENCE"); raw != "" {
		if value, err := strconv.ParseFloat(raw, 32); err == nil {
			cfg.WakeMinConfidence = float32(value)
		}
	}
}

func envString(name string, target *string) {
	if value := os.Getenv(name); value != "" {
		*target = value
	}
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}
