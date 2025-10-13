package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	an "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/meedamian/fat/internal/types"
	"github.com/openai/openai-go"
	oa "github.com/openai/openai-go/option"
	"google.golang.org/genai"
)

var ModelMap = map[string]*types.ModelInfo{
	"A": {ID: "A", Name: "grok-4-fast", MaxTok: 131072, BaseURL: "https://api.x.ai/v1/chat/completions"},
	"B": {ID: "B", Name: "gpt-5-mini", MaxTok: 16384, BaseURL: "https://api.openai.com/v1/chat/completions"},
	"C": {ID: "C", Name: "claude-3.5-haiku", MaxTok: 200000, BaseURL: "https://api.anthropic.com/v1/messages"},
	"D": {ID: "D", Name: "gemini-2.5-flash", MaxTok: 128000, BaseURL: "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent"},
}

var DefaultRates = map[string]types.Rate{
	"grok-4-fast":      {TS: 0, In: 0.20, Out: 0.50},
	"gpt-5-mini":       {TS: 0, In: 0.25, Out: 2.00},
	"claude-3.5-haiku": {TS: 0, In: 0.80, Out: 4.00},
	"gemini-2.5-flash": {TS: 0, In: 0.35, Out: 1.05},
}

// InitClients initializes SDK clients for each model
func InitClients(rates map[string]types.Rate) {
	for _, mi := range ModelMap {
		if rate, ok := rates[mi.Name]; ok {
			mi.Rates = rate
		} else if def, ok := DefaultRates[mi.Name]; ok {
			mi.Rates = def
		}
		switch mi.Name {
		case "grok-4-fast":
			// No client for Grok
		case "gpt-5-mini":
			mi.Client = openai.NewClient(oa.WithAPIKey(mi.APIKey), oa.WithMaxRetries(3))
		case "claude-3.5-haiku":
			mi.Client = anthropic.NewClient(an.WithAPIKey(mi.APIKey), an.WithMaxRetries(3))
		case "gemini-2.5-flash":
			client, _ := genai.NewClient(context.Background(), &genai.ClientConfig{APIKey: mi.APIKey})
			mi.Client = client
		}
	}
}

// CallModel calls the model with prompt and history, returns Response, tokens
func CallModel(ctx context.Context, mi *types.ModelInfo, prompt string, history []string) (types.Response, int64, int64, error) {
	var resp types.Response
	var tokIn, tokOut int64
	switch mi.Name {
	case "grok-4-fast":
		return callGrok(ctx, mi, prompt, history)
	case "gpt-5-mini":
		return callOpenAI(ctx, mi, prompt, history)
	case "claude-3.5-haiku":
		return callClaude(ctx, mi, prompt, history)
	case "gemini-2.5-flash":
		return callGemini(ctx, mi, prompt, history)
	}
	return resp, tokIn, tokOut, fmt.Errorf("unknown model")
}

func callGrok(ctx context.Context, mi *types.ModelInfo, prompt string, history []string) (types.Response, int64, int64, error) {
	var resp types.Response
	var tokIn, tokOut int64
	messages := []map[string]string{{"role": "user", "content": prompt}}
	for _, h := range history {
		messages = append(messages, map[string]string{"role": "assistant", "content": h})
	}
	body := map[string]interface{}{
		"model":    mi.Name,
		"messages": messages,
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", mi.BaseURL, bytes.NewBuffer(jsonBody))
	req.Header.Set("Authorization", "Bearer "+mi.APIKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return types.Response{}, 0, 0, err
	}
	defer res.Body.Close()
	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)
	if res.StatusCode != 200 {
		return types.Response{}, 0, 0, fmt.Errorf("grok error: %v", result)
	}
	content := result["choices"].([]interface{})[0].(map[string]interface{})["message"].(map[string]interface{})["content"].(string)
	usage := result["usage"].(map[string]interface{})
	tokIn = int64(usage["prompt_tokens"].(float64))
	tokOut = int64(usage["completion_tokens"].(float64))
	// Parse JSON response
	json.Unmarshal([]byte(content), &resp)
	return resp, tokIn, tokOut, nil
}

func callOpenAI(ctx context.Context, mi *types.ModelInfo, prompt string, history []string) (types.Response, int64, int64, error) {
	var resp types.Response
	var tokIn, tokOut int64
	client := mi.Client.(*openai.Client)
	params := openai.ChatCompletionNewParams{
		Model: openai.ChatModel("gpt-5-mini"),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(prompt),
		},
	}
	for _, h := range history {
		params.Messages = append(params.Messages, openai.AssistantMessage(h))
	}
	result, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		return types.Response{}, 0, 0, err
	}
	content := result.Choices[0].Message.Content
	json.Unmarshal([]byte(content), &resp)
	tokIn = result.Usage.PromptTokens
	tokOut = result.Usage.CompletionTokens
	return resp, tokIn, tokOut, nil
}

func callClaude(ctx context.Context, mi *types.ModelInfo, prompt string, history []string) (types.Response, int64, int64, error) {
	var resp types.Response
	var tokIn, tokOut int64
	client := mi.Client.(*anthropic.Client)
	// schema := jsonschema.Reflect(&types.Response{})
	// tool := anthropic.ToolParam{
	// 	Name:        "response",
	// 	Description: "Provide refined answer and suggestions",
	// 	InputSchema: schema,
	// }
	params := anthropic.MessageNewParams{
		Model:     anthropic.Model("claude-3.5-haiku"),
		MaxTokens: 1024,
		System:    []anthropic.TextBlockParam{{Text: "Respond with JSON: {\"refined\": \"...\", \"suggestions\": [...]}"} },
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
		// Tools: []anthropic.ToolParam{tool},
	}
	for _, h := range history {
		params.Messages = append(params.Messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(h)))
	}
	result, err := client.Messages.New(ctx, params)
	if err != nil {
		return types.Response{}, 0, 0, err
	}
	// Assume tool use
	content := result.Content[0].Text
	// For simplicity, assume direct text, but spec says tool
	// TODO: handle tool properly
	json.Unmarshal([]byte(content), &resp)
	tokIn = result.Usage.InputTokens
	tokOut = result.Usage.OutputTokens
	return resp, tokIn, tokOut, nil
}

func callGemini(ctx context.Context, mi *types.ModelInfo, prompt string, history []string) (types.Response, int64, int64, error) {
	var resp types.Response
	var tokIn, tokOut int64
	client := mi.Client.(*genai.Client)
	contents := []*genai.Content{{Role: genai.RoleUser, Parts: []*genai.Part{{Text: prompt}}}}
	for _, h := range history {
		contents = append(contents, &genai.Content{Role: genai.RoleModel, Parts: []*genai.Part{{Text: h}}})
	}
	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
	}
	result, err := client.Models.GenerateContent(ctx, mi.Name, contents, config)
	if err != nil {
		return types.Response{}, 0, 0, err
	}
	content := result.Candidates[0].Content.Parts[0].Text
	json.Unmarshal([]byte(content), &resp)
	tokIn = int64(result.UsageMetadata.PromptTokenCount)
	tokOut = int64(result.UsageMetadata.CandidatesTokenCount)
	return resp, tokIn, tokOut, nil
}

// CostForToks calculates cost
func CostForToks(mi *types.ModelInfo, tokIn, tokOut int64) float64 {
	return mi.Rates.In*float64(tokIn) + mi.Rates.Out*float64(tokOut)
}
