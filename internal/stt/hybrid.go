package stt

import (
	"context"
	"fmt"
	"strings"

	"assistance/internal/audio"
)

// HybridTranscriber tries local STT first and falls back to cloud STT when confidence is low.
type HybridTranscriber struct {
	local          Transcriber
	cloud          Transcriber
	localLanguages []string
	cloudLanguages []string
	minConfidence  float32
}

// NewHybridTranscriber creates a local-first STT router.
func NewHybridTranscriber(local Transcriber, cloud Transcriber, localLanguages []string, cloudLanguages []string, minConfidence float32) *HybridTranscriber {
	if minConfidence <= 0 {
		minConfidence = 0.65
	}
	return &HybridTranscriber{
		local:          local,
		cloud:          cloud,
		localLanguages: localLanguages,
		cloudLanguages: cloudLanguages,
		minConfidence:  minConfidence,
	}
}

// Transcribe tries Vosk-compatible local STT first, then retries with cloud STT for Darija/Moroccan Arabic.
func (t *HybridTranscriber) Transcribe(ctx context.Context, clip audio.AudioClip, fallbackLanguages []string) (Transcript, error) {
	localLanguages := firstNonEmpty(t.localLanguages, fallbackLanguages)
	localTranscript, localErr := t.local.Transcribe(ctx, clip, localLanguages)
	if localErr == nil && shouldAcceptLocal(localTranscript, t.minConfidence) {
		return localTranscript, nil
	}

	cloudLanguages := firstNonEmpty(t.cloudLanguages, fallbackLanguages)
	cloudTranscript, cloudErr := t.cloud.Transcribe(ctx, clip, cloudLanguages)
	if cloudErr == nil {
		return cloudTranscript, nil
	}
	if localErr != nil {
		return Transcript{}, fmt.Errorf("hybrid STT failed locally (%v) and in cloud (%w)", localErr, cloudErr)
	}
	return Transcript{}, fmt.Errorf("hybrid STT local confidence %.2f below %.2f and cloud failed: %w", localTranscript.Confidence, t.minConfidence, cloudErr)
}

func shouldAcceptLocal(transcript Transcript, minConfidence float32) bool {
	if strings.TrimSpace(transcript.Text) == "" {
		return false
	}
	if transcript.Confidence == 0 {
		return true
	}
	return transcript.Confidence >= minConfidence
}

func firstNonEmpty(primary []string, fallback []string) []string {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}
