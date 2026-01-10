package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/meedamian/fat/internal/shared"
	"github.com/meedamian/fat/internal/types"
)

const (
	Grok = "grok"

	Grok41Fast             = "grok-4-1-fast"
	Grok41FastNonReasoning = "grok-4-1-fast-non-reasoning"
	Grok4Fast              = "grok-4-fast"
	Grok4FastNonReasoning  = "grok-4-fast-non-reasoning"
	GrokCodeFast1          = "grok-code-fast-1"
	Grok4                  = "grok-4"
	Grok3Mini              = "grok-3-mini"
	Grok3                  = "grok-3"
)

// Models list: https://docs.x.ai/docs/models
var GrokFamily = types.ModelFamily{
	ID:       Grok,
	Provider: "xAI",
	BaseURL:  "https://api.x.ai/v1/chat/completions",
	Variants: map[string]types.ModelVariant{
		Grok41Fast:             {MaxTok: 2_000_000, Rate: types.Rate{In: 0.2, Out: 0.5}},
		Grok41FastNonReasoning: {MaxTok: 2_000_000, Rate: types.Rate{In: 0.2, Out: 0.5}},
		Grok4Fast:              {MaxTok: 2_000_000, Rate: types.Rate{In: 0.2, Out: 0.5}},
		Grok4FastNonReasoning:  {MaxTok: 2_000_000, Rate: types.Rate{In: 0.2, Out: 0.5}},
		GrokCodeFast1:          {MaxTok: 256_000, Rate: types.Rate{In: 0.2, Out: 1.5}},
		Grok4:                  {MaxTok: 256_000, Rate: types.Rate{In: 3.0, Out: 15.0}},
		Grok3Mini:              {MaxTok: 131_072, Rate: types.Rate{In: 0.3, Out: 0.5}},
		Grok3:                  {MaxTok: 131_072, Rate: types.Rate{In: 3.0, Out: 15.0}},
	},
}

// GrokModel implements the Model interface for Grok
type GrokModel struct {
	info   *types.ModelInfo
	client *http.Client
}

// NewGrokModel creates a new Grok model instance
func NewGrokModel(info *types.ModelInfo) *GrokModel {
	return &GrokModel{
		info:   info,
		client: shared.NewHTTPClient(info.RequestTimeout),
	}
}

// grokResponse represents the API response structure
type grokResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
	} `json:"usage"`
}

// Prompt implements the Model interface
func (m *GrokModel) Prompt(ctx context.Context, question string, meta types.Meta, replies map[string]types.Reply, discussion map[string]map[string][]types.DiscussionMessage, privateNotes map[int]string) (types.ModelResult, error) {
	prompt := shared.FormatPrompt(m.info.ID, m.info.Name, question, meta, replies, discussion, privateNotes)

	// Build messages array
	messages := []map[string]string{{"role": "user", "content": prompt}}

	// Call Grok API
	body := map[string]any{
		"model":    m.info.Name,
		"messages": messages,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return types.ModelResult{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", m.info.BaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return types.ModelResult{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+m.info.APIKey)
	req.Header.Set("Content-Type", "application/json")

	res, err := m.client.Do(req)
	if err != nil {
		return types.ModelResult{}, fmt.Errorf("api request failed: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return types.ModelResult{}, fmt.Errorf("api returned status %d", res.StatusCode)
	}

	var result grokResponse
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return types.ModelResult{}, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Choices) == 0 {
		return types.ModelResult{}, fmt.Errorf("no choices in response")
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
