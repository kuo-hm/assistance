package realtime

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"assistance/internal/memory"
	"assistance/internal/wakeword"
	"golang.org/x/sync/errgroup"
)

// RunnerConfig controls a realtime session.
type RunnerConfig struct {
	InputCommand  string
	OutputCommand string
	InputRate     int
	OutputRate    int
	ChunkBytes    int
	Languages     []string
}

// MemoryStore is the memory surface used by realtime mode.
type MemoryStore interface {
	LoadContext(ctx context.Context, transcript string) (memory.Context, error)
	SaveTurn(ctx context.Context, sessionID string, role string, language string, text string) error
	UpdateSummary(ctx context.Context, sessionID string) error
}

// SessionFactory creates a new realtime session per wake event.
type SessionFactory func() (Session, error)

// Runner owns wake events and realtime audio loops.
type Runner struct {
	wake    wakeword.Detector
	memory  MemoryStore
	factory SessionFactory
	config  RunnerConfig
}

// NewRunner creates a realtime runner.
func NewRunner(wake wakeword.Detector, memory MemoryStore, factory SessionFactory, config RunnerConfig) *Runner {
	return &Runner{wake: wake, memory: memory, factory: factory, config: config}
}

// Run waits for wake events and starts realtime sessions.
func (r *Runner) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		wakeCtx, wakeCancel := context.WithCancel(ctx)
		events, err := r.wake.Listen(wakeCtx)
		if err != nil {
			wakeCancel()
			return err
		}

		select {
		case <-ctx.Done():
			wakeCancel()
			return ctx.Err()
		case event, ok := <-events:
			// Cancel wake detection immediately so it releases the microphone device lock
			wakeCancel()
			if !ok {
				return nil
			}
			sessionID := fmt.Sprintf("%d", event.DetectedAt.UnixNano())
			slog.Info("realtime wake word detected", "phrase", event.Phrase, "session_id", sessionID)
			if err := r.runSession(ctx, sessionID); err != nil {
				if errors.Is(err, context.Canceled) {
					continue
				}
				slog.Warn("realtime session ended", "error", err)
			}
		}
	}
}

func (r *Runner) runSession(parent context.Context, sessionID string) error {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()

	mem, err := r.memory.LoadContext(ctx, "")
	if err != nil {
		return fmt.Errorf("load realtime memory: %w", err)
	}
	session, err := r.factory()
	if err != nil {
		return err
	}
	defer session.Close()
	if err := session.Connect(ctx, mem); err != nil {
		return err
	}

	input, err := StartInputStream(ctx, r.config.InputCommand, r.config.InputRate)
	if err != nil {
		return err
	}
	defer input.Close()

	player := NewOutputPlayer(r.config.OutputCommand, r.config.OutputRate)
	defer func() {
		if err := player.Close(); err != nil {
			slog.Debug("close realtime player", "error", err)
		}
	}()
	var stopOnce sync.Once
	stopSession := func() {
		stopOnce.Do(func() {
			cancel()
			_ = session.Close()
			_ = input.Close()
			_ = player.Close()
		})
	}
	group, groupCtx := errgroup.WithContext(ctx)

	group.Go(func() error {
		reader := NewChunkReader(input.Reader(), r.config.ChunkBytes)
		for {
			chunk, err := reader.ReadChunk()
			if err != nil {
				stopSession()
				return fmt.Errorf("read realtime audio input: %w", err)
			}
			if err := session.SendAudio(groupCtx, chunk); err != nil {
				stopSession()
				return err
			}
		}
	})

	group.Go(func() error {
		for {
			event, err := session.Receive(groupCtx)
			if err != nil {
				stopSession()
				return err
			}
			if event.InputText != "" {
				fmt.Println("you:", event.InputText)
				if err := r.memory.SaveTurn(groupCtx, sessionID, "user", "", event.InputText); err != nil {
					stopSession()
					return err
				}
			}
			if event.OutputText != "" {
				fmt.Println("assistant:", event.OutputText)
				if err := r.memory.SaveTurn(groupCtx, sessionID, "assistant", "", event.OutputText); err != nil {
					stopSession()
					return err
				}
				if err := r.memory.UpdateSummary(groupCtx, sessionID); err != nil {
					stopSession()
					return err
				}
			}
			if len(event.Audio) > 0 {
				if err := player.Play(groupCtx, event.Audio); err != nil {
					stopSession()
					return err
				}
			}
			if event.Interrupted {
				slog.Info("realtime output interrupted")
			}
			if event.TurnComplete || event.GenerationDone {
				slog.Debug("realtime model turn completed", "time", time.Now())
			}
		}
	})

	if err := group.Wait(); err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return err
	}
	return nil
}
