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
	DeepSeek = "deepseek"

	DeepSeekChat  = "deepseek-chat"
	DeepSeekCoder = "deepseek-coder"
)

// DeepSeekModel implements the Model interface for DeepSeek
type DeepSeekModel struct {
	info   *types.ModelInfo
	client openai.Client
}

// NewDeepSeekModel creates a new DeepSeek model instance
func NewDeepSeekModel(info *types.ModelInfo) *DeepSeekModel {
	client := openai.NewClient(
		oa.WithAPIKey(info.APIKey),
		oa.WithBaseURL(info.BaseURL),
		oa.WithMaxRetries(3),
	)
	return &DeepSeekModel{
		info:   info,
		client: client,
	}
}

// Prompt implements the Model interface
func (m *DeepSeekModel) Prompt(ctx context.Context, question string, meta types.Meta, replies map[string]types.Reply, discussion map[string]map[string][]types.DiscussionMessage, privateNotes map[int]string) (types.ModelResult, error) {
	prompt := shared.FormatPrompt(m.info.ID, m.info.Name, question, meta, replies, discussion, privateNotes)

	params := openai.ChatCompletionNewParams{
		Model: openai.ChatModel(m.info.Name),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	}

	result, err := m.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return types.ModelResult{}, fmt.Errorf("deepseek api call failed: %w", err)
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
