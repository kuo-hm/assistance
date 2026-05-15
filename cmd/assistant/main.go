package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"assistance/internal/audio"
	"assistance/internal/config"
	"assistance/internal/consoleio"
	"assistance/internal/conversation"
	"assistance/internal/llm"
	"assistance/internal/memory"
	"assistance/internal/stt"
	"assistance/internal/tts"
	"assistance/internal/wakeword"
)

func main() {
	if err := run(); err != nil {
		slog.Error("assistant stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	store, err := memory.Open(cfg.SQLitePath)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			slog.Warn("close memory store", "error", closeErr)
		}
	}()

	consoleLines := consoleio.NewLineReader(os.Stdin)

	detector, err := buildWakeDetector(cfg, consoleLines)
	if err != nil {
		return err
	}
	recorder := audio.NewExternalRecorder(cfg.RecordCommand, cfg.RecordSeconds)
	transcriber, err := buildTranscriber(cfg, consoleLines)
	if err != nil {
		return err
	}
	assistant, err := buildAssistant(cfg)
	if err != nil {
		return err
	}
	speaker, err := buildSpeaker(cfg)
	if err != nil {
		return err
	}

	runner := conversation.NewRunner(conversation.Dependencies{
		WakeWordDetector: detector,
		Recorder:         recorder,
		Transcriber:      transcriber,
		Assistant:        assistant,
		Speaker:          speaker,
		Memory:           store,
		Config: conversation.Config{
			Languages:          cfg.Languages,
			SilenceTimeout:     cfg.SilenceTimeout,
			SessionIdleTimeout: cfg.SessionIdleTimeout,
			MaxTurnsPerSession: cfg.MaxTurnsPerSession,
		},
	})

	fmt.Println("assistant ready; waiting for wake phrase")
	return runner.Run(ctx)
}

func buildAssistant(cfg config.Config) (llm.Assistant, error) {
	switch cfg.LLMProvider {
	case "gemini":
		return llm.NewGeminiAssistant(cfg.GeminiAPIKey, cfg.GeminiModel)
	case "openai":
		return llm.NewOpenAIAssistant(cfg.OpenAIAPIKey, cfg.OpenAIModel)
	default:
		return nil, fmt.Errorf("unsupported llm provider %q", cfg.LLMProvider)
	}
}

func buildWakeDetector(cfg config.Config, consoleLines *consoleio.LineReader) (wakeword.Detector, error) {
	switch cfg.WakeProvider {
	case "console":
		return wakeword.NewConsoleDetectorWithReader(consoleLines, os.Stdout, cfg.WakePhrase), nil
	case "command":
		return wakeword.NewCommandDetector(cfg.WakeCommand, cfg.WakePhrase), nil
	default:
		return nil, fmt.Errorf("unsupported wake provider %q", cfg.WakeProvider)
	}
}

func buildTranscriber(cfg config.Config, consoleLines *consoleio.LineReader) (stt.Transcriber, error) {
	if !cfg.RecordingEnabled {
		return stt.NewConsoleTranscriberWithReader(consoleLines, os.Stdout), nil
	}

	switch cfg.STTProvider {
	case "console":
		return stt.NewConsoleTranscriberWithReader(consoleLines, os.Stdout), nil
	case "google":
		return stt.NewGoogleTranscriber(context.Background(), cfg.GoogleCredentialsFile)
	case "vosk":
		return stt.NewVoskTranscriber(cfg.VoskModelPath, cfg.VoskSampleRate)
	case "hybrid":
		local, err := stt.NewVoskTranscriber(cfg.VoskModelPath, cfg.VoskSampleRate)
		if err != nil {
			return nil, err
		}
		cloud, err := stt.NewGoogleTranscriber(context.Background(), cfg.GoogleCredentialsFile)
		if err != nil {
			return nil, err
		}
		return stt.NewHybridTranscriber(
			local,
			cloud,
			cfg.HybridLocalLanguages,
			cfg.HybridCloudLanguages,
			cfg.HybridMinConfidence,
		), nil
	default:
		return nil, fmt.Errorf("unsupported stt provider %q", cfg.STTProvider)
	}
}

func buildSpeaker(cfg config.Config) (tts.Speaker, error) {
	console := tts.NewConsoleSpeaker(os.Stdout)
	switch cfg.TTSProvider {
	case "console":
		return console, nil
	case "google":
		if cfg.PlayCommand == "" {
			return nil, errors.New("ASSISTANT_PLAY_COMMAND is required when ASSISTANT_TTS_PROVIDER=google")
		}
		googleSpeaker, err := tts.NewGoogleSpeaker(context.Background(), tts.GoogleSpeakerConfig{
			CredentialsFile: cfg.GoogleCredentialsFile,
			LanguageCode:    cfg.TTSLanguageCode,
			VoiceName:       cfg.TTSVoiceName,
			PlayCommand:     cfg.PlayCommand,
		})
		if err != nil {
			return nil, err
		}
		return tts.NewMultiSpeaker(console, googleSpeaker), nil
	case "windows":
		return tts.NewMultiSpeaker(console, tts.NewWindowsSpeaker()), nil
	default:
		return nil, fmt.Errorf("unsupported tts provider %q", cfg.TTSProvider)
	}
}
