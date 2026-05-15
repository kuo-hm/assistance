package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"assistance/internal/memory"
	"google.golang.org/genai"
)

// ConversationInput is one user turn plus memory.
type ConversationInput struct {
	SessionID string
	UserText  string
	Language  string
	Memory    memory.Context
}

// AssistantReply is the model response.
type AssistantReply struct {
	Text     string
	Language string
}

// Assistant generates assistant replies.
type Assistant interface {
	Generate(ctx context.Context, input ConversationInput) (AssistantReply, error)
}

// GeminiAssistant uses the official Google GenAI Go SDK.
type GeminiAssistant struct {
	client *genai.Client
	model  string
}

// NewGeminiAssistant creates a Gemini-backed assistant.
func NewGeminiAssistant(apiKey string, model string) (*GeminiAssistant, error) {
	if apiKey == "" {
		return nil, errors.New("gemini api key is required")
	}
	if model == "" {
		model = "gemini-2.5-flash"
	}
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fmt.Errorf("create gemini client: %w", err)
	}
	return &GeminiAssistant{client: client, model: model}, nil
}

// Generate sends the turn and memory to Gemini.
func (a *GeminiAssistant) Generate(ctx context.Context, input ConversationInput) (AssistantReply, error) {
	prompt := BuildPrompt(input)
	resp, err := a.client.Models.GenerateContent(ctx, a.model, genai.Text(prompt), &genai.GenerateContentConfig{
		SystemInstruction: &genai.Content{Parts: []*genai.Part{{Text: systemInstruction}}},
		Temperature:       genai.Ptr[float32](0.6),
	})
	if err != nil {
		return AssistantReply{}, fmt.Errorf("generate gemini response: %w", err)
	}
	text := strings.TrimSpace(responseText(resp))
	if text == "" {
		return AssistantReply{}, errors.New("gemini returned an empty response")
	}
	return AssistantReply{Text: text, Language: input.Language}, nil
}

// BuildPrompt formats memory and the current user turn.
func BuildPrompt(input ConversationInput) string {
	var builder strings.Builder
	builder.WriteString("Conversation memory:\n")
	if input.Memory.Summary != "" {
		builder.WriteString(input.Memory.Summary)
		builder.WriteByte('\n')
	} else {
		builder.WriteString("No summary yet.\n")
	}
	if len(input.Memory.Facts) > 0 {
		builder.WriteString("Known user facts:\n")
		for key, value := range input.Memory.Facts {
			builder.WriteString("- ")
			builder.WriteString(key)
			builder.WriteString(": ")
			builder.WriteString(value)
			builder.WriteByte('\n')
		}
	}
	if len(input.Memory.Recent) > 0 {
		builder.WriteString("Recent turns:\n")
		for _, turn := range input.Memory.Recent {
			builder.WriteString(turn.Role)
			builder.WriteString(": ")
			builder.WriteString(turn.Text)
			builder.WriteByte('\n')
		}
	}
	builder.WriteString("\nUser language hint: ")
	builder.WriteString(input.Language)
	builder.WriteString("\nUser said:\n")
	builder.WriteString(input.UserText)
	return builder.String()
}

const systemInstruction = `You are a voice assistant for one user.
Understand English, French, and Moroccan Darija. Moroccan Darija may be written in Arabic script or Latin chat spelling.
Reply naturally and briefly, usually in the same language as the user.
If the user mixes languages, you may mix English, French, and Darija in a natural Moroccan style.
Avoid long markdown because the answer will be spoken aloud.
If useful user preferences appear, mention them clearly so the memory layer can store them later.`

func responseText(resp *genai.GenerateContentResponse) string {
	if resp == nil {
		return ""
	}
	var builder strings.Builder
	for _, candidate := range resp.Candidates {
		if candidate == nil || candidate.Content == nil {
			continue
		}
		for _, part := range candidate.Content.Parts {
			if part != nil && part.Text != "" {
				builder.WriteString(part.Text)
			}
		}
	}
	return builder.String()
}
