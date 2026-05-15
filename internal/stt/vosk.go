//go:build vosk

package stt

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"assistance/internal/audio"
	vosk "github.com/alphacep/vosk-api/go"
)

// VoskTranscriber uses the offline Vosk recognizer.
type VoskTranscriber struct {
	model      *vosk.VoskModel
	sampleRate float64
}

// NewVoskTranscriber creates an offline Vosk STT adapter.
func NewVoskTranscriber(modelPath string, sampleRate float64) (*VoskTranscriber, error) {
	if strings.TrimSpace(modelPath) == "" {
		return nil, errors.New("ASSISTANT_VOSK_MODEL_PATH is required when ASSISTANT_STT_PROVIDER=vosk")
	}
	if sampleRate <= 0 {
		sampleRate = 16000
	}
	vosk.SetLogLevel(-1)
	model, err := vosk.NewModel(modelPath)
	if err != nil {
		return nil, fmt.Errorf("load vosk model: %w", err)
	}
	return &VoskTranscriber{model: model, sampleRate: sampleRate}, nil
}

// Transcribe converts 16 kHz mono PCM/WAV audio to text locally.
func (t *VoskTranscriber) Transcribe(ctx context.Context, clip audio.AudioClip, languages []string) (Transcript, error) {
	select {
	case <-ctx.Done():
		return Transcript{}, ctx.Err()
	default:
	}
	if len(clip.Data) == 0 {
		return Transcript{}, errors.New("vosk requires recorded audio data")
	}

	pcm, wavSampleRate, err := pcmFromWAV(clip.Data)
	if err != nil {
		return Transcript{}, err
	}
	sampleRate := t.sampleRate
	if wavSampleRate > 0 {
		sampleRate = float64(wavSampleRate)
	}

	recognizer, err := vosk.NewRecognizer(t.model, sampleRate)
	if err != nil {
		return Transcript{}, fmt.Errorf("create vosk recognizer: %w", err)
	}
	defer recognizer.Free()

	const chunkSize = 4096
	for start := 0; start < len(pcm); start += chunkSize {
		select {
		case <-ctx.Done():
			return Transcript{}, ctx.Err()
		default:
		}
		end := start + chunkSize
		if end > len(pcm) {
			end = len(pcm)
		}
		recognizer.AcceptWaveform(pcm[start:end])
	}

	var result voskResult
	if err := json.Unmarshal([]byte(recognizer.FinalResult()), &result); err != nil {
		return Transcript{}, fmt.Errorf("parse vosk result: %w", err)
	}
	text := strings.TrimSpace(result.Text)
	if text == "" {
		return Transcript{}, errors.New("vosk returned no transcript")
	}

	language := ""
	if len(languages) > 0 {
		language = languages[0]
	}
	return Transcript{Text: text, Language: language, Confidence: resultConfidence(result)}, nil
}

type voskResult struct {
	Text   string     `json:"text"`
	Result []voskWord `json:"result"`
}

type voskWord struct {
	Confidence float32 `json:"conf"`
}

func resultConfidence(result voskResult) float32 {
	if len(result.Result) == 0 {
		return 0
	}
	var total float32
	for _, word := range result.Result {
		total += word.Confidence
	}
	return total / float32(len(result.Result))
}
