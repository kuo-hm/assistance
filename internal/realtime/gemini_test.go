package realtime

import (
	"testing"

	"google.golang.org/genai"
)

func TestEventFromGemini(t *testing.T) {
	message := &genai.LiveServerMessage{
		ServerContent: &genai.LiveServerContent{
			TurnComplete:       true,
			GenerationComplete: true,
			InputTranscription: &genai.Transcription{
				Text:     " hello ",
				Finished: true,
			},
			OutputTranscription: &genai.Transcription{
				Text:     " bonjour ",
				Finished: true,
			},
			ModelTurn: &genai.Content{
				Parts: []*genai.Part{
					{InlineData: &genai.Blob{MIMEType: "audio/pcm", Data: []byte{1, 2, 3}}},
				},
			},
		},
	}

	event := eventFromGemini(message)
	if event.InputText != "hello" {
		t.Fatalf("InputText = %q", event.InputText)
	}
	if event.OutputText != "bonjour" {
		t.Fatalf("OutputText = %q", event.OutputText)
	}
	if !event.TurnComplete || !event.GenerationDone {
		t.Fatalf("completion flags = turn:%v generation:%v", event.TurnComplete, event.GenerationDone)
	}
	if string(event.Audio) != string([]byte{1, 2, 3}) {
		t.Fatalf("Audio = %v", event.Audio)
	}
}
