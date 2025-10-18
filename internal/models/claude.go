package models

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
	an "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/meedamian/fat/internal/shared"
	"github.com/meedamian/fat/internal/types"
)

// ClaudeModel implements the Model interface for Anthropic Claude
type ClaudeModel struct {
	info   *types.ModelInfo
	client anthropic.Client
}

// NewClaudeModel creates a new Claude model instance
func NewClaudeModel(info *types.ModelInfo) *ClaudeModel {
	client := anthropic.NewClient(an.WithAPIKey(info.APIKey), an.WithMaxRetries(3))
	return &ClaudeModel{
		info:   info,
		client: client,
	}
}

// Prompt implements the Model interface
func (m *ClaudeModel) Prompt(ctx context.Context, question string, meta types.Meta, replies map[string]string, discussion map[string][]string) (types.ModelResult, error) {
	prompt := shared.FormatPrompt("Claude", question, meta, replies, discussion)

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model("claude-3-5-haiku-latest"),
		MaxTokens: 1024,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	}

	result, err := m.client.Messages.New(ctx, params)
	if err != nil {
		return types.ModelResult{}, err
	}

	content := result.Content[0].Text
	reply := shared.ParseResponse(content)

	return types.ModelResult{
		Reply:  reply,
		TokIn:  result.Usage.InputTokens,
		TokOut: result.Usage.OutputTokens,
		Prompt: prompt,
	}, nil
}
