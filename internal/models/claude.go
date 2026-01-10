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

	Claude45Opus   = "claude-opus-4-5"
	Claude45Sonnet = "claude-sonnet-4-5"
	Claude45Haiku  = "claude-haiku-4-5"
	Claude41Opus   = "claude-opus-4-1"
	Claude4Sonnet  = "claude-sonnet-4-0"
	Claude37Sonnet = "claude-3-7-sonnet-latest"
	Claude4Opus    = "claude-opus-4-0"
	Claude35Haiku  = "claude-3-5-haiku-latest"
)

// Models list: https://docs.claude.com/en/docs/about-claude/models/overview
var ClaudeFamily = types.ModelFamily{
	ID:       Claude,
	Provider: "Anthropic",
	BaseURL:  "https://api.anthropic.com/v1/messages",
	Variants: map[string]types.ModelVariant{
		// NOTE: Claude Sonnet 4.5 supports a 1M token context window when using the context-1m-2025-08-07 beta header. Long context pricing applies to requests exceeding 200K tokens.
		// NOTE: Claude Sonnet 4 supports a 1M token context window when using the context-1m-2025-08-07 beta header. Long context pricing applies to requests exceeding 200K tokens.
		Claude45Opus:   {MaxTok: 200_000, Rate: types.Rate{In: 5.0, Out: 25.0}},
		Claude45Sonnet: {MaxTok: 200_000, Rate: types.Rate{In: 3.0, Out: 15.0}},
		Claude45Haiku:  {MaxTok: 200_000, Rate: types.Rate{In: 1.0, Out: 5.0}},
		Claude41Opus:   {MaxTok: 200_000, Rate: types.Rate{In: 15.0, Out: 75.0}},
		Claude4Sonnet:  {MaxTok: 200_000, Rate: types.Rate{In: 3.0, Out: 15.0}},
		Claude37Sonnet: {MaxTok: 200_000, Rate: types.Rate{In: 3.0, Out: 15.0}},
		Claude4Opus:    {MaxTok: 200_000, Rate: types.Rate{In: 15.0, Out: 75.0}},
		Claude35Haiku:  {MaxTok: 200_000, Rate: types.Rate{In: 0.8, Out: 4.0}},
	},
}

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
func (m *ClaudeModel) Prompt(ctx context.Context, question string, meta types.Meta, replies map[string]types.Reply, discussion map[string]map[string][]types.DiscussionMessage, privateNotes map[int]string) (types.ModelResult, error) {
	prompt := shared.FormatPrompt(m.info.ID, m.info.Name, question, meta, replies, discussion, privateNotes)

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
