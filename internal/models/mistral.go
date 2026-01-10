package models

import (
	"context"
	"fmt"

	"github.com/meedamian/fat/internal/shared"
	"github.com/meedamian/fat/internal/types"
	"github.com/openai/openai-go"
	oa "github.com/openai/openai-go/option"
)

const (
	Mistral = "mistral"

	MagistralMedium = "magistral-medium-latest"
	MistralLarge    = "mistral-large-latest"
	MistralMedium   = "mistral-medium-latest"
	MistralSmall    = "mistral-small-latest"
	Codestral       = "codestral-latest"
	Ministral3B     = "ministral-3b-latest"
	Ministral8B     = "ministral-8b-latest"
)

// Models list: https://docs.mistral.ai/getting-started/models/
// Pricing: https://mistral.ai/technology/#pricing
var MistralFamily = types.ModelFamily{
	ID:       Mistral,
	Provider: "Mistral AI",
	BaseURL:  "https://api.mistral.ai/v1",
	Variants: map[string]types.ModelVariant{
		MagistralMedium: {MaxTok: 128_000, Rate: types.Rate{In: 2.0, Out: 5.0}},
		MistralLarge:    {MaxTok: 256_000, Rate: types.Rate{In: 0.5, Out: 1.5}},
		MistralMedium:   {MaxTok: 128_000, Rate: types.Rate{In: 0.4, Out: 2.0}},
		MistralSmall:    {MaxTok: 32_000, Rate: types.Rate{In: 0.1, Out: 0.3}},
		Codestral:       {MaxTok: 256_000, Rate: types.Rate{In: 0.3, Out: 0.9}},
		Ministral3B:     {MaxTok: 128_000, Rate: types.Rate{In: 0.04, Out: 0.04}},
		Ministral8B:     {MaxTok: 128_000, Rate: types.Rate{In: 0.1, Out: 0.1}},
	},
}

// MistralModel implements the Model interface for Mistral AI
type MistralModel struct {
	info   *types.ModelInfo
	client openai.Client
}

// NewMistralModel creates a new Mistral model instance
func NewMistralModel(info *types.ModelInfo) *MistralModel {
	// Mistral uses OpenAI-compatible API
	client := openai.NewClient(
		oa.WithAPIKey(info.APIKey),
		oa.WithBaseURL("https://api.mistral.ai/v1"),
		oa.WithMaxRetries(3),
	)
	return &MistralModel{
		info:   info,
		client: client,
	}
}

// Prompt implements the Model interface
func (m *MistralModel) Prompt(ctx context.Context, question string, meta types.Meta, replies map[string]types.Reply, discussion map[string]map[string][]types.DiscussionMessage, privateNotes map[int]string) (types.ModelResult, error) {
	prompt := shared.FormatPrompt(m.info.ID, m.info.Name, question, meta, replies, discussion, privateNotes)

	params := openai.ChatCompletionNewParams{
		Model: openai.ChatModel(m.info.Name),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	}

	result, err := m.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return types.ModelResult{}, fmt.Errorf("mistral api call failed: %w", err)
	}

	content := result.Choices[0].Message.Content
	reply := shared.ParseResponse(content)

	return types.ModelResult{
		Reply:  reply,
		TokIn:  result.Usage.PromptTokens,
		TokOut: result.Usage.CompletionTokens,
		Prompt: prompt,
	}, nil
}
