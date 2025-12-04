package models

import (
	"context"
	"fmt"

	"github.com/meedamian/fat/internal/shared"
	"github.com/meedamian/fat/internal/types"
	"google.golang.org/genai"
)

const (
	Gemini = "gemini"

	Gemini3ProPreview = "gemini-3-pro-preview"
	Gemini25Pro       = "gemini-2.5-pro"
	Gemini25Flash     = "gemini-2.5-flash"
	Gemini25FlashLite = "gemini-2.5-flash-lite"
	Gemini20Flash     = "gemini-2.0-flash"
	Gemini20FlashLite = "gemini-2.0-flash-lite"
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
func (m *GeminiModel) Prompt(ctx context.Context, question string, meta types.Meta, replies map[string]types.Reply, discussion map[string]map[string][]types.DiscussionMessage, privateNotes map[int]string) (types.ModelResult, error) {
	if m.client == nil {
		return types.ModelResult{}, fmt.Errorf("gemini client not initialized")
	}

	prompt := shared.FormatPrompt(m.info.ID, m.info.Name, question, meta, replies, discussion, privateNotes)

	result, err := m.client.Models.GenerateContent(ctx, m.info.Name, genai.Text(prompt), nil)
	if err != nil {
		return types.ModelResult{}, fmt.Errorf("gemini api call failed: %w", err)
	}

	content := result.Text()
	reply := shared.ParseResponse(content)

	// Extract token usage from UsageMetadata
	var tokIn, tokOut int64
	if result.UsageMetadata != nil {
		tokIn = int64(result.UsageMetadata.PromptTokenCount)
		tokOut = int64(result.UsageMetadata.CandidatesTokenCount)
	}

	return types.ModelResult{
		Reply:  reply,
		TokIn:  tokIn,
		TokOut: tokOut,
		Prompt: prompt,
	}, nil
}
