package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/meedamian/fat/internal/shared"
	"github.com/meedamian/fat/internal/types"
)

// GrokModel implements the Model interface for Grok
type GrokModel struct {
	info *types.ModelInfo
}

// NewGrokModel creates a new Grok model instance
func NewGrokModel(info *types.ModelInfo) *GrokModel {
	return &GrokModel{info: info}
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
func (m *GrokModel) Prompt(ctx context.Context, question string, meta types.Meta, replies map[string]types.Reply, discussion map[string][]string) (types.ModelResult, error) {
	prompt := shared.FormatPrompt(m.info.Name, question, meta, replies, discussion)

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

	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return types.ModelResult{}, fmt.Errorf("api request failed: %w", err)
	}
	defer res.Body.Close()

	var result grokResponse
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return types.ModelResult{}, fmt.Errorf("failed to decode response: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		return types.ModelResult{}, fmt.Errorf("api returned status %d", res.StatusCode)
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
