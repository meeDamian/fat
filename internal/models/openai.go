package models

import (
	"context"

	"github.com/meedamian/fat/internal/shared"
	"github.com/meedamian/fat/internal/types"
	"github.com/openai/openai-go"
	oa "github.com/openai/openai-go/option"
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
func (m *OpenAIModel) Prompt(ctx context.Context, question string, replies map[string]string, discussion map[string][]string) (types.ModelResult, error) {
	prompt := shared.FormatPrompt("GPT", question, replies, discussion)

	params := openai.ChatCompletionNewParams{
		Model: openai.ChatModel("gpt-5-mini"),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	}

	result, err := m.client.Chat.Completions.New(ctx, params)
	if err != nil {
		return types.ModelResult{}, err
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
