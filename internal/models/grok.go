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

// Prompt implements the Model interface
func (m *GrokModel) Prompt(ctx context.Context, question string, meta types.Meta, replies map[string]string, discussion map[string][]string) (types.ModelResult, error) {
	prompt := shared.FormatPrompt("Grok", question, meta, replies, discussion)

	// Build messages array
	messages := []map[string]string{{"role": "user", "content": prompt}}

	// Call Grok API
	body := map[string]any{
		"model":    m.info.Name,
		"messages": messages,
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", m.info.BaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return types.ModelResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+m.info.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return types.ModelResult{}, err
	}
	defer res.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return types.ModelResult{}, err
	}

	if res.StatusCode != 200 {
		return types.ModelResult{}, fmt.Errorf("grok API error: %v", result)
	}

	content := result["choices"].([]any)[0].(map[string]any)["message"].(map[string]any)["content"].(string)
	usage := result["usage"].(map[string]any)
	tokIn := int64(usage["prompt_tokens"].(float64))
	tokOut := int64(usage["completion_tokens"].(float64))

	reply := shared.ParseResponse(content)

	return types.ModelResult{
		Reply:  reply,
		TokIn:  tokIn,
		TokOut: tokOut,
		Prompt: prompt,
	}, nil
}
