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

	Gemini31Pro       = "gemini-3.1-pro-preview"
	Gemini31FlashLite = "gemini-3.1-flash-lite-preview"
	Gemini3Pro        = "gemini-3-pro-preview"
	Gemini3Flash      = "gemini-3-flash-preview"
	Gemini25Pro       = "gemini-2.5-pro"
	Gemini25Flash     = "gemini-2.5-flash"
	Gemini25FlashLite = "gemini-2.5-flash-lite"
	Gemini20Flash     = "gemini-2.0-flash"
	Gemini20FlashLite = "gemini-2.0-flash-lite"
)

// Models list: https://ai.google.dev/gemini-api/docs/models
var GeminiFamily = types.ModelFamily{
	ID:       Gemini,
	Provider: "Google",
	BaseURL:  "https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent", // Updated to placeholder for flexibility.
	Variants: map[string]types.ModelVariant{
		Gemini31Pro:       {MaxTok: 1_048_576, Rate: types.Rate{In: 2.0, Out: 12.0}},
		Gemini31FlashLite: {MaxTok: 1_048_576, Rate: types.Rate{In: 0.25, Out: 1.5}},

		Gemini3Pro:   {MaxTok: 1_048_576, Rate: types.Rate{In: 2.0, Out: 12.0}},
		Gemini3Flash: {MaxTok: 1_048_576, Rate: types.Rate{In: 0.5, Out: 3.0}},

		Gemini25Pro:       {MaxTok: 1_048_576, Rate: types.Rate{In: 1.25, Out: 10.0}},
		Gemini25Flash:     {MaxTok: 1_048_576, Rate: types.Rate{In: 0.3, Out: 2.5}},
		Gemini25FlashLite: {MaxTok: 1_048_576, Rate: types.Rate{In: 0.1, Out: 0.4}},

		Gemini20Flash:     {MaxTok: 1_048_576, Rate: types.Rate{In: 0.1, Out: 0.4}},
		Gemini20FlashLite: {MaxTok: 1_048_576, Rate: types.Rate{In: 0.075, Out: 0.3}},
	},
}

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
