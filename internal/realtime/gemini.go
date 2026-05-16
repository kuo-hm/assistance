package realtime

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"assistance/internal/memory"
	"google.golang.org/genai"
)

// GeminiConfig configures Gemini Live.
type GeminiConfig struct {
	APIKey      string
	Model       string
	Voice       string
	Languages   []string
	Temperature float32
}

type liveSession interface {
	SendRealtimeInput(input genai.LiveRealtimeInput) error
	Receive() (*genai.LiveServerMessage, error)
	Close() error
}

// GeminiSession uses Gemini Live API for speech-to-speech.
type GeminiSession struct {
	cfg     GeminiConfig
	client  *genai.Client
	session liveSession
}

// NewGeminiSession creates a Gemini realtime session.
func NewGeminiSession(cfg GeminiConfig) (*GeminiSession, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("gemini api key is required")
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = "gemini-2.5-flash-native-audio-preview-12-2025"
	}
	if strings.TrimSpace(cfg.Voice) == "" {
		cfg.Voice = "Kore"
	}
	if cfg.Temperature == 0 {
		cfg.Temperature = 0.6
	}
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  cfg.APIKey,
		Backend: genai.BackendGeminiAPI,
		HTTPOptions: genai.HTTPOptions{
			APIVersion: "v1alpha",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create gemini realtime client: %w", err)
	}
	return &GeminiSession{cfg: cfg, client: client}, nil
}

// Connect opens the Gemini Live WebSocket and injects memory via system instruction.
func (s *GeminiSession) Connect(ctx context.Context, mem memory.Context) error {
	config := &genai.LiveConnectConfig{
		ResponseModalities: []genai.Modality{genai.ModalityAudio},
		Temperature:        genai.Ptr(s.cfg.Temperature),
		ThinkingConfig:     &genai.ThinkingConfig{ThinkingBudget: genai.Ptr(int32(0))},
		SystemInstruction:  &genai.Content{Parts: []*genai.Part{{Text: buildRealtimeInstruction(mem)}}},
		SpeechConfig: &genai.SpeechConfig{
			VoiceConfig: &genai.VoiceConfig{
				PrebuiltVoiceConfig: &genai.PrebuiltVoiceConfig{VoiceName: s.cfg.Voice},
			},
		},
		InputAudioTranscription:  &genai.AudioTranscriptionConfig{},
		OutputAudioTranscription: &genai.AudioTranscriptionConfig{},
		RealtimeInputConfig: &genai.RealtimeInputConfig{
			AutomaticActivityDetection: &genai.AutomaticActivityDetection{},
		},
	}
	session, err := s.client.Live.Connect(ctx, s.cfg.Model, config)
	if err != nil {
		return fmt.Errorf("connect gemini live: %w", err)
	}
	s.session = session
	return nil
}

// SendAudio sends one PCM chunk to Gemini.
func (s *GeminiSession) SendAudio(_ context.Context, chunk []byte) error {
	if s.session == nil {
		return errors.New("gemini realtime session is not connected")
	}
	return s.session.SendRealtimeInput(genai.LiveRealtimeInput{
		Audio: &genai.Blob{Data: chunk, MIMEType: "audio/pcm"},
	})
}

// Receive waits for the next Gemini Live event.
func (s *GeminiSession) Receive(_ context.Context) (Event, error) {
	if s.session == nil {
		return Event{}, errors.New("gemini realtime session is not connected")
	}
	message, err := s.session.Receive()
	if err != nil {
		return Event{}, err
	}
	return eventFromGemini(message), nil
}

// Close closes the Gemini session.
func (s *GeminiSession) Close() error {
	if s.session == nil {
		return nil
	}
	return s.session.Close()
}

func eventFromGemini(message *genai.LiveServerMessage) Event {
	var event Event
	if message == nil || message.ServerContent == nil {
		return event
	}
	content := message.ServerContent
	event.TurnComplete = content.TurnComplete
	event.Interrupted = content.Interrupted
	event.GenerationDone = content.GenerationComplete
	if content.InputTranscription != nil && content.InputTranscription.Finished {
		event.InputText = strings.TrimSpace(content.InputTranscription.Text)
	}
	if content.OutputTranscription != nil && content.OutputTranscription.Finished {
		event.OutputText = strings.TrimSpace(content.OutputTranscription.Text)
	}
	if content.ModelTurn != nil {
		for _, part := range content.ModelTurn.Parts {
			if part == nil {
				continue
			}
			if part.Text != "" {
				if event.OutputText != "" {
					event.OutputText += " "
				}
				event.OutputText += strings.TrimSpace(part.Text)
			}
			if part.InlineData != nil && strings.HasPrefix(part.InlineData.MIMEType, "audio/") {
				event.Audio = append(event.Audio, part.InlineData.Data...)
			}
		}
	}
	return event
}

func buildRealtimeInstruction(mem memory.Context) string {
	var builder strings.Builder
	builder.WriteString("You are a realtime voice assistant for one user. ")
	builder.WriteString("Understand English, French, and Moroccan Darija. Reply briefly and naturally, usually in the user's language. ")
	builder.WriteString("Only say the final answer to the user. Do not describe your reasoning, planning, analysis, or clarification process. ")
	builder.WriteString("Never use markdown, bullets, headings, or formatting because replies are spoken aloud.\n")
	if mem.Summary != "" {
		builder.WriteString("\nMemory summary:\n")
		builder.WriteString(mem.Summary)
	}
	if len(mem.Facts) > 0 {
		builder.WriteString("\nKnown facts:\n")
		for key, value := range mem.Facts {
			builder.WriteString("- ")
			builder.WriteString(key)
			builder.WriteString(": ")
			builder.WriteString(value)
			builder.WriteByte('\n')
		}
	}
	if len(mem.Recent) > 0 {
		builder.WriteString("\nRecent turns:\n")
		for _, turn := range mem.Recent {
			builder.WriteString(turn.Role)
			builder.WriteString(": ")
			builder.WriteString(turn.Text)
			builder.WriteByte('\n')
		}
	}
	return builder.String()
}
