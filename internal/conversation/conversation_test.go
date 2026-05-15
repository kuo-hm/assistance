package conversation

import (
	"context"
	"errors"
	"testing"
	"time"

	"assistance/internal/audio"
	"assistance/internal/llm"
	"assistance/internal/memory"
	"assistance/internal/stt"
	"assistance/internal/wakeword"
)

func TestRunnerProcessesOneSessionTurn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wakeEvents := make(chan wakeword.WakeEvent, 1)
	wakeEvents <- wakeword.WakeEvent{Phrase: "hello there", DetectedAt: time.Now()}
	close(wakeEvents)

	mem := &fakeMemory{}
	runner := NewRunner(Dependencies{
		WakeWordDetector: fakeWakeDetector{events: wakeEvents},
		Recorder:         fakeRecorder{},
		Transcriber:      &fakeTranscriber{texts: []string{"hello", "stop"}},
		Assistant:        fakeAssistant{},
		Speaker:          &fakeSpeaker{},
		Memory:           mem,
		Config: Config{
			Languages:          []string{"en-US", "fr-FR", "ar-MA"},
			SilenceTimeout:     time.Second,
			SessionIdleTimeout: time.Second,
			MaxTurnsPerSession: 3,
		},
	})

	if err := runner.Run(ctx); err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(mem.turns) != 2 {
		t.Fatalf("saved turn count = %d", len(mem.turns))
	}
	if !mem.summaryUpdated {
		t.Fatal("expected summary update")
	}
}

type fakeWakeDetector struct {
	events <-chan wakeword.WakeEvent
}

func (d fakeWakeDetector) Listen(context.Context) (<-chan wakeword.WakeEvent, error) {
	return d.events, nil
}

type fakeRecorder struct{}

func (r fakeRecorder) RecordUntilSilence(context.Context, audio.RecordOptions) (audio.AudioClip, error) {
	return audio.AudioClip{Data: []byte("audio"), MIMEType: "audio/wav", SampleRateHertz: 16000}, nil
}

type fakeTranscriber struct {
	texts []string
}

func (t *fakeTranscriber) Transcribe(context.Context, audio.AudioClip, []string) (stt.Transcript, error) {
	if len(t.texts) == 0 {
		return stt.Transcript{}, errors.New("no transcript")
	}
	text := t.texts[0]
	t.texts = t.texts[1:]
	return stt.Transcript{Text: text, Language: "en-US", Confidence: 1}, nil
}

type fakeAssistant struct{}

func (a fakeAssistant) Generate(context.Context, llm.ConversationInput) (llm.AssistantReply, error) {
	return llm.AssistantReply{Text: "hi", Language: "en-US"}, nil
}

type fakeSpeaker struct{}

func (s *fakeSpeaker) Speak(context.Context, llm.AssistantReply) error {
	return nil
}

type fakeMemory struct {
	turns          []string
	summaryUpdated bool
}

func (m *fakeMemory) LoadContext(context.Context, string) (memory.Context, error) {
	return memory.Context{}, nil
}

func (m *fakeMemory) SaveTurn(_ context.Context, _ string, role string, _ string, text string) error {
	m.turns = append(m.turns, role+":"+text)
	return nil
}

func (m *fakeMemory) UpdateSummary(context.Context, string) error {
	m.summaryUpdated = true
	return nil
}
