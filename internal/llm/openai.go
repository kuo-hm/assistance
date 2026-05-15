package llm

import (
	"context"
	"errors"
	"fmt"
	"strings"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/openai/openai-go/responses"
	"github.com/openai/openai-go/shared"
)

// OpenAIAssistant uses OpenAI's Responses API.
type OpenAIAssistant struct {
	client openai.Client
	model  string
}

// NewOpenAIAssistant creates an OpenAI-backed assistant.
func NewOpenAIAssistant(apiKey string, model string) (*OpenAIAssistant, error) {
	if apiKey == "" {
		return nil, errors.New("openai api key is required")
	}
	if model == "" {
		model = "gpt-4.1-mini"
	}
	client := openai.NewClient(option.WithAPIKey(apiKey))
	return &OpenAIAssistant{client: client, model: model}, nil
}

// Generate sends the turn and memory to OpenAI.
func (a *OpenAIAssistant) Generate(ctx context.Context, input ConversationInput) (AssistantReply, error) {
	prompt := BuildPrompt(input)
	resp, err := a.client.Responses.New(ctx, responses.ResponseNewParams{
		Model: shared.ResponsesModel(a.model),
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(prompt),
		},
		Instructions: openai.String(systemInstruction),
		Temperature:  openai.Float(0.6),
	})
	if err != nil {
		return AssistantReply{}, fmt.Errorf("generate openai response: %w", err)
	}
	text := strings.TrimSpace(resp.OutputText())
	if text == "" {
		return AssistantReply{}, errors.New("openai returned an empty response")
	}
	return AssistantReply{Text: text, Language: input.Language}, nil
}
