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
	WakeProvider          string        `json:"wake_provider"`
	WakePhrase            string        `json:"wake_phrase"`
	WakeCommand           string        `json:"wake_command"`
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
}

type fileConfig struct {
	GeminiAPIKey          string   `json:"gemini_api_key"`
	GeminiModel           string   `json:"gemini_model"`
	LLMProvider           string   `json:"llm_provider"`
	OpenAIAPIKey          string   `json:"openai_api_key"`
	OpenAIModel           string   `json:"openai_model"`
	GoogleCredentialsFile string   `json:"google_credentials_file"`
	WakeProvider          string   `json:"wake_provider"`
	WakePhrase            string   `json:"wake_phrase"`
	WakeCommand           string   `json:"wake_command"`
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

	return cfg, nil
}

func defaults() Config {
	return Config{
		GeminiModel:      "gemini-2.5-flash",
		LLMProvider:      "gemini",
		OpenAIModel:      "gpt-4.1-mini",
		WakeProvider:     "console",
		WakePhrase:       "hello there",
		STTProvider:      "console",
		TTSProvider:      "console",
		SQLitePath:       "assistant.db",
		RecordingEnabled: false,
		Languages:        []string{"en-US", "fr-FR", "ar-MA"},
		VoskSampleRate:   16000,
		HybridLocalLanguages: []string{
			"en-US",
			"fr-FR",
		},
		HybridCloudLanguages: []string{"ar-MA"},
		HybridMinConfidence:  0.65,
		SilenceTimeout:       3 * time.Second,
		SessionIdleTimeout:   45 * time.Second,
		MaxTurnsPerSession:   12,
		RecordSeconds:        12,
		TTSLanguageCode:      "ar-XA",
		TTSVoiceName:         "",
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
	if fc.WakeProvider != "" {
		cfg.WakeProvider = fc.WakeProvider
	}
	if fc.WakePhrase != "" {
		cfg.WakePhrase = fc.WakePhrase
	}
	if fc.WakeCommand != "" {
		cfg.WakeCommand = fc.WakeCommand
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
	return nil
}

func applyEnv(cfg *Config) {
	envString("GEMINI_API_KEY", &cfg.GeminiAPIKey)
	envString("GEMINI_MODEL", &cfg.GeminiModel)
	envString("ASSISTANT_LLM_PROVIDER", &cfg.LLMProvider)
	envString("OPENAI_API_KEY", &cfg.OpenAIAPIKey)
	envString("OPENAI_MODEL", &cfg.OpenAIModel)
	envString("GOOGLE_APPLICATION_CREDENTIALS", &cfg.GoogleCredentialsFile)
	envString("ASSISTANT_WAKE_PROVIDER", &cfg.WakeProvider)
	envString("ASSISTANT_WAKE_PHRASE", &cfg.WakePhrase)
	envString("ASSISTANT_WAKE_COMMAND", &cfg.WakeCommand)
	envString("ASSISTANT_STT_PROVIDER", &cfg.STTProvider)
	envString("ASSISTANT_TTS_PROVIDER", &cfg.TTSProvider)
	envString("ASSISTANT_SQLITE_PATH", &cfg.SQLitePath)
	envString("ASSISTANT_RECORD_COMMAND", &cfg.RecordCommand)
	envString("ASSISTANT_PLAY_COMMAND", &cfg.PlayCommand)
	envString("ASSISTANT_TTS_LANGUAGE_CODE", &cfg.TTSLanguageCode)
	envString("ASSISTANT_TTS_VOICE_NAME", &cfg.TTSVoiceName)
	envString("ASSISTANT_VOSK_MODEL_PATH", &cfg.VoskModelPath)

	if raw := os.Getenv("ASSISTANT_RECORDING_ENABLED"); raw != "" {
		if value, err := strconv.ParseBool(raw); err == nil {
			cfg.RecordingEnabled = value
		}
	}
	if raw := os.Getenv("ASSISTANT_LANGUAGES"); raw != "" {
		cfg.Languages = splitCSV(raw)
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
