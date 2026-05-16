package realtime

import (
	"context"

	"assistance/internal/memory"
)

// Event is one realtime provider event.
type Event struct {
	InputText      string
	OutputText     string
	Audio          []byte
	TurnComplete   bool
	Interrupted    bool
	GenerationDone bool
}

// Session is a realtime speech-to-speech model session.
type Session interface {
	Connect(ctx context.Context, memory memory.Context) error
	SendAudio(ctx context.Context, chunk []byte) error
	Receive(ctx context.Context) (Event, error)
	Close() error
}
