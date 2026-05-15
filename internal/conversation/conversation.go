package conversation

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"assistance/internal/audio"
	"assistance/internal/llm"
	"assistance/internal/memory"
	"assistance/internal/stt"
	"assistance/internal/tts"
	"assistance/internal/wakeword"
)

// Config controls the conversation loop.
type Config struct {
	Languages          []string
	SilenceTimeout     time.Duration
	SessionIdleTimeout time.Duration
	MaxTurnsPerSession int
}

// Dependencies are the assistant subsystem interfaces.
type Dependencies struct {
	WakeWordDetector wakeword.Detector
	Recorder         audio.Recorder
	Transcriber      stt.Transcriber
	Assistant        llm.Assistant
	Speaker          tts.Speaker
	Memory           interface {
		LoadContext(ctx context.Context, transcript string) (memory.Context, error)
		SaveTurn(ctx context.Context, sessionID string, role string, language string, text string) error
		UpdateSummary(ctx context.Context, sessionID string) error
	}
	Config Config
}

// Runner owns the standby and continuous conversation loops.
type Runner struct {
	deps Dependencies
}

// NewRunner creates a conversation runner.
func NewRunner(deps Dependencies) *Runner {
	return &Runner{deps: deps}
}

// Run waits for wake events and starts conversation sessions.
func (r *Runner) Run(ctx context.Context) error {
	if err := r.validate(); err != nil {
		return err
	}
	events, err := r.deps.WakeWordDetector.Listen(ctx)
	if err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-events:
			if !ok {
				return nil
			}
			sessionID := fmt.Sprintf("%d", event.DetectedAt.UnixNano())
			slog.Info("wake word detected", "phrase", event.Phrase, "session_id", sessionID)
			if err := r.runSession(ctx, sessionID); err != nil && !errors.Is(err, context.Canceled) {
				slog.Warn("conversation session ended with error", "error", err)
			}
		}
	}
}

func (r *Runner) runSession(parent context.Context, sessionID string) error {
	sessionCtx, cancel := context.WithTimeout(parent, r.deps.Config.SessionIdleTimeout)
	defer cancel()

	for turn := 0; turn < r.deps.Config.MaxTurnsPerSession; turn++ {
		clip, err := r.deps.Recorder.RecordUntilSilence(sessionCtx, audio.RecordOptions{
			SilenceTimeout: r.deps.Config.SilenceTimeout,
		})
		if err != nil {
			return fmt.Errorf("record audio: %w", err)
		}

		transcript, err := r.deps.Transcriber.Transcribe(sessionCtx, clip, r.deps.Config.Languages)
		if err != nil {
			return fmt.Errorf("transcribe audio: %w", err)
		}
		if transcript.Text == "exit" || transcript.Text == "stop" || transcript.Text == "quit" {
			return nil
		}

		mem, err := r.deps.Memory.LoadContext(sessionCtx, transcript.Text)
		if err != nil {
			return fmt.Errorf("load memory: %w", err)
		}
		if err := r.deps.Memory.SaveTurn(sessionCtx, sessionID, "user", transcript.Language, transcript.Text); err != nil {
			return fmt.Errorf("save user turn: %w", err)
		}

		reply, err := r.deps.Assistant.Generate(sessionCtx, llm.ConversationInput{
			SessionID: sessionID,
			UserText:  transcript.Text,
			Language:  transcript.Language,
			Memory:    mem,
		})
		if err != nil {
			return fmt.Errorf("generate reply: %w", err)
		}
		if err := r.deps.Memory.SaveTurn(sessionCtx, sessionID, "assistant", reply.Language, reply.Text); err != nil {
			return fmt.Errorf("save assistant turn: %w", err)
		}
		if err := r.deps.Speaker.Speak(sessionCtx, reply); err != nil {
			return fmt.Errorf("speak reply: %w", err)
		}
		if err := r.deps.Memory.UpdateSummary(sessionCtx, sessionID); err != nil {
			return fmt.Errorf("update summary: %w", err)
		}
	}
	return nil
}

func (r *Runner) validate() error {
	if r.deps.WakeWordDetector == nil {
		return errors.New("wake word detector is required")
	}
	if r.deps.Recorder == nil {
		return errors.New("recorder is required")
	}
	if r.deps.Transcriber == nil {
		return errors.New("transcriber is required")
	}
	if r.deps.Assistant == nil {
		return errors.New("assistant is required")
	}
	if r.deps.Speaker == nil {
		return errors.New("speaker is required")
	}
	if r.deps.Memory == nil {
		return errors.New("memory is required")
	}
	if len(r.deps.Config.Languages) == 0 {
		return errors.New("languages are required")
	}
	if r.deps.Config.SessionIdleTimeout <= 0 {
		return errors.New("session idle timeout must be positive")
	}
	if r.deps.Config.MaxTurnsPerSession <= 0 {
		return errors.New("max turns per session must be positive")
	}
	return nil
}
