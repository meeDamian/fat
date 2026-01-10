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
	GPT = "gpt"

	GPT52    = "gpt-5.2"
	GPT52Pro = "gpt-5.2-pro"

	GPT51         = "gpt-5.1"
	GPT51Codex    = "gpt-5.1-codex"
	GPT51CodexMax = "gpt-5.1-codex-max"

	GPT5Pro   = "gpt-5-pro"
	GPT5      = "gpt-5"
	GPT5Mini  = "gpt-5-mini"
	GPT5Nano  = "gpt-5-nano"
	GPT5Codex = "gpt-5-codex"

	GPT41     = "gpt-4.1"
	GPT41Mini = "gpt-4.1-mini"
	GPT41Nano = "gpt-4.1-nano"
)

// Models list: https://platform.openai.com/docs/models
var GPTFamily = types.ModelFamily{
	ID:       GPT,
	Provider: "OpenAI",
	BaseURL:  "https://api.openai.com/v1/chat/completions",
	Variants: map[string]types.ModelVariant{
		GPT52:    {MaxTok: 400_000, Rate: types.Rate{In: 1.75, Out: 14.0}},
		GPT52Pro: {MaxTok: 400_000, Rate: types.Rate{In: 21.0, Out: 168.0}},

		GPT51:         {MaxTok: 400_000, Rate: types.Rate{In: 1.25, Out: 10.0}},
		GPT51Codex:    {MaxTok: 400_000, Rate: types.Rate{In: 1.25, Out: 10.0}},
		GPT51CodexMax: {MaxTok: 400_000, Rate: types.Rate{In: 1.25, Out: 10.0}},

		GPT5Pro:   {MaxTok: 400_000, Rate: types.Rate{In: 15.0, Out: 120.0}},
		GPT5:      {MaxTok: 400_000, Rate: types.Rate{In: 1.25, Out: 10.0}},
		GPT5Codex: {MaxTok: 400_000, Rate: types.Rate{In: 1.25, Out: 10.0}},
		GPT5Mini:  {MaxTok: 400_000, Rate: types.Rate{In: 0.25, Out: 2.0}},
		GPT5Nano:  {MaxTok: 400_000, Rate: types.Rate{In: 0.05, Out: 0.4}},

		GPT41:     {MaxTok: 1_047_576, Rate: types.Rate{In: 2.0, Out: 8.0}},
		GPT41Mini: {MaxTok: 1_047_576, Rate: types.Rate{In: 0.4, Out: 1.6}},
		GPT41Nano: {MaxTok: 1_047_576, Rate: types.Rate{In: 0.1, Out: 0.4}},
	},
}

// OpenAIModel implements the Model interface for OpenAI
type OpenAIModel struct {
	info   *types.ModelInfo
	client openai.Client
}

// NewOpenAIModel creates a new OpenAI model instance
func NewOpenAIModel(info *types.ModelInfo) *OpenAIModel {
	client := openai.NewClient(oa.WithAPIKey(info.APIKey), oa.WithMaxRetries(3))
	return &OpenAIModel{
		info:   info,
		client: client,
	}
}

// Prompt implements the Model interface
func (m *OpenAIModel) Prompt(ctx context.Context, question string, meta types.Meta, replies map[string]types.Reply, discussion map[string]map[string][]types.DiscussionMessage, privateNotes map[int]string) (types.ModelResult, error) {
	prompt := shared.FormatPrompt(m.info.ID, m.info.Name, question, meta, replies, discussion, privateNotes)

	params := openai.ChatCompletionNewParams{
		Model: openai.ChatModel(m.info.Name),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	}

	result, err := m.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return types.ModelResult{}, fmt.Errorf("openai api call failed: %w", err)
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
