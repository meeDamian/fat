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

	GPT51     = "gpt-5.1"
	GPT5Pro   = "gpt-5-pro"
	GPT5      = "gpt-5"
	GPT5Mini  = "gpt-5-mini"
	GPT5Nano  = "gpt-5-nano"
	GPT5Codex = "gpt-5-codex"
	GPT41     = "gpt-4.1"
	GPT41Mini = "gpt-4.1-mini"
	GPT41Nano = "gpt-4.1-nano"
)

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
func (m *OpenAIModel) Prompt(ctx context.Context, question string, meta types.Meta, replies map[string]types.Reply, discussion map[string]map[string][]types.DiscussionMessage) (types.ModelResult, error) {
	prompt := shared.FormatPrompt(m.info.ID, m.info.Name, question, meta, replies, discussion)

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
