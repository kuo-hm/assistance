package stt

import (
	"context"
	"errors"
	"testing"

	"assistance/internal/audio"
)

func TestHybridTranscriberUsesLocalWhenConfident(t *testing.T) {
	local := &fakeTranscriber{transcript: Transcript{Text: "hello", Language: "en-US", Confidence: 0.9}}
	cloud := &fakeTranscriber{transcript: Transcript{Text: "salam", Language: "ar-MA", Confidence: 0.9}}
	hybrid := NewHybridTranscriber(local, cloud, []string{"en-US"}, []string{"ar-MA"}, 0.65)

	got, err := hybrid.Transcribe(context.Background(), audio.AudioClip{Data: []byte("audio")}, nil)
	if err != nil {
		t.Fatalf("Transcribe() error = %v", err)
	}
	if got.Text != "hello" {
		t.Fatalf("Text = %q", got.Text)
	}
	if cloud.calls != 0 {
		t.Fatalf("cloud calls = %d", cloud.calls)
	}
}

func TestHybridTranscriberFallsBackWhenLocalLowConfidence(t *testing.T) {
	local := &fakeTranscriber{transcript: Transcript{Text: "wrong", Language: "en-US", Confidence: 0.3}}
	cloud := &fakeTranscriber{transcript: Transcript{Text: "salam", Language: "ar-MA", Confidence: 0.8}}
	hybrid := NewHybridTranscriber(local, cloud, []string{"en-US"}, []string{"ar-MA"}, 0.65)

	got, err := hybrid.Transcribe(context.Background(), audio.AudioClip{Data: []byte("audio")}, nil)
	if err != nil {
		t.Fatalf("Transcribe() error = %v", err)
	}
	if got.Text != "salam" {
		t.Fatalf("Text = %q", got.Text)
	}
	if cloud.calls != 1 {
		t.Fatalf("cloud calls = %d", cloud.calls)
	}
}

func TestHybridTranscriberReportsBothErrors(t *testing.T) {
	local := &fakeTranscriber{err: errors.New("local failed")}
	cloud := &fakeTranscriber{err: errors.New("cloud failed")}
	hybrid := NewHybridTranscriber(local, cloud, []string{"en-US"}, []string{"ar-MA"}, 0.65)

	_, err := hybrid.Transcribe(context.Background(), audio.AudioClip{Data: []byte("audio")}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
}

type fakeTranscriber struct {
	transcript Transcript
	err        error
	calls      int
}

func (t *fakeTranscriber) Transcribe(context.Context, audio.AudioClip, []string) (Transcript, error) {
	t.calls++
	if t.err != nil {
		return Transcript{}, t.err
	}
	return t.transcript, nil
}
