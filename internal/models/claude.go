package models

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	an "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/meedamian/fat/internal/shared"
	"github.com/meedamian/fat/internal/types"
)

const (
	Claude = "claude"

	Claude45Sonnet = "claude-sonnet-4-5"
	Claude45Haiku  = "claude-haiku-4-5"
	Claude41Opus   = "claude-opus-4-1"
	Claude4Sonnet  = "claude-sonnet-4-0"
	Claude37Sonnet = "claude-3-7-sonnet-latest"
	Claude4Opus    = "claude-opus-4-0"
	Claude35Haiku  = "claude-3-5-haiku-latest"
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
func (m *ClaudeModel) Prompt(ctx context.Context, question string, meta types.Meta, replies map[string]types.Reply, discussion map[string]map[string][]types.DiscussionMessage) (types.ModelResult, error) {
	prompt := shared.FormatPrompt(m.info.ID, m.info.Name, question, meta, replies, discussion)

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(m.info.Name),
		MaxTokens: 1024,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	}

	result, err := m.client.Messages.New(ctx, params)
	if err != nil {
		return types.ModelResult{}, fmt.Errorf("claude api call failed: %w", err)
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
