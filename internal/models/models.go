package models

import (
	"github.com/meedamian/fat/internal/types"
)

var ModelMap = map[string]*types.ModelInfo{
	"grok":   {ID: "grok", Name: "grok-4-fast", MaxTok: 131072, BaseURL: "https://api.x.ai/v1/chat/completions"},
	"gpt":    {ID: "gpt", Name: "gpt-5-mini", MaxTok: 16384, BaseURL: "https://api.openai.com/v1/chat/completions"},
	"claude": {ID: "claude", Name: "claude-3-5-haiku-20241022", MaxTok: 200000, BaseURL: "https://api.anthropic.com/v1/messages"},
	"gemini": {ID: "gemini", Name: "gemini-2.5-flash", MaxTok: 128000, BaseURL: "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent"},
}

// NewModel creates a Model implementation for the given model info
func NewModel(info *types.ModelInfo) types.Model {
	switch info.ID {
	case "grok":
		return NewGrokModel(info)
	case "gpt":
		return NewOpenAIModel(info)
	case "claude":
		return NewClaudeModel(info)
	case "gemini":
		return NewGeminiModel(info)
	default:
		return nil
	}
}
