package models

import (
	"context"
	"fmt"

	"github.com/meedamian/fat/internal/shared"
	"github.com/meedamian/fat/internal/types"
	"google.golang.org/genai"
)

// GeminiModel implements the Model interface for Google Gemini
type GeminiModel struct {
	info   *types.ModelInfo
	client *genai.Client
}

// NewGeminiModel creates a new Gemini model instance
func NewGeminiModel(info *types.ModelInfo) *GeminiModel {
	client, err := genai.NewClient(context.Background(), &genai.ClientConfig{APIKey: info.APIKey})
	if err != nil {
		// Log error but return model anyway - error will surface on first Prompt call
		if info.Logger != nil {
			info.Logger.Error("failed to create gemini client", "error", err)
		}
	}
	return &GeminiModel{
		info:   info,
		client: client,
	}
}

// Prompt implements the Model interface
func (m *GeminiModel) Prompt(ctx context.Context, question string, meta types.Meta, replies map[string]string, discussion map[string][]string) (types.ModelResult, error) {
	if m.client == nil {
		return types.ModelResult{}, fmt.Errorf("gemini client not initialized")
	}

	prompt := shared.FormatPrompt(m.info.Name, question, meta, replies, discussion)

	result, err := m.client.Models.GenerateContent(ctx, m.info.Name, genai.Text(prompt), nil)
	if err != nil {
		return types.ModelResult{}, fmt.Errorf("gemini api call failed: %w", err)
	}

	content := result.Text()
	reply := shared.ParseResponse(content)

	// Gemini doesn't provide token usage, so we estimate or set to 0
	return types.ModelResult{
		Reply:  reply,
		TokIn:  0, // Not available from Gemini API
		TokOut: 0, // Not available from Gemini API
		Prompt: prompt,
	}, nil
}
