package wakeword

import (
	"context"
	"testing"
	"time"

	"assistance/internal/audio"
	"assistance/internal/stt"
)

func TestNormalizePhrase(t *testing.T) {
	got := normalizePhrase(" Hello,   THERE! ")
	if got != "hello there" {
		t.Fatalf("normalizePhrase() = %q", got)
	}
}

func TestVoiceDetectorEmitsWakeEvent(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	detector := NewVoiceDetector(fakeRecorder{}, &fakeTranscriber{text: "yes hello there please"}, VoiceConfig{
		Phrase:    "hello there",
		Languages: []string{"en-US"},
	})

	events, err := detector.Listen(ctx)
	if err != nil {
		t.Fatalf("Listen() error = %v", err)
	}

	select {
	case event := <-events:
		if event.Phrase != "hello there" {
			t.Fatalf("Phrase = %q", event.Phrase)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for wake event")
	}
}

func TestMatchesAnyPhraseUsesAliases(t *testing.T) {
	targets := normalizePhrases("wakeup", []string{"wake up", "up"})
	if !matchesAnyPhrase("up", targets) {
		t.Fatal("matchesAnyPhrase() did not match alias")
	}
}

type fakeRecorder struct{}

func (fakeRecorder) RecordUntilSilence(context.Context, audio.RecordOptions) (audio.AudioClip, error) {
	return audio.AudioClip{Data: []byte{1}, MIMEType: "audio/wav", SampleRateHertz: 16000}, nil
}

type fakeTranscriber struct {
	text string
}

func (t *fakeTranscriber) Transcribe(context.Context, audio.AudioClip, []string) (stt.Transcript, error) {
	return stt.Transcript{Text: t.text, Confidence: 1}, nil
}
