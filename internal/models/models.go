package models

import (
	"github.com/meedamian/fat/internal/types"
)

var ModelMap = map[string]*types.ModelInfo{
	Grok:   {ID: Grok, Name: Grok4Fast, MaxTok: 131072, BaseURL: "https://api.x.ai/v1/chat/completions"},
	GPT:    {ID: GPT, Name: GPT5Mini, MaxTok: 16384, BaseURL: "https://api.openai.com/v1/chat/completions"},
	Claude: {ID: Claude, Name: Claude35Haiku, MaxTok: 200000, BaseURL: "https://api.anthropic.com/v1/messages"},
	Gemini: {ID: Gemini, Name: Gemini25Flash, MaxTok: 128000, BaseURL: "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent"},
}

// NewModel creates a Model implementation for the given model info
func NewModel(info *types.ModelInfo) types.Model {
	switch info.ID {
	case Grok:
		return NewGrokModel(info)
	case GPT:
		return NewOpenAIModel(info)
	case Claude:
		return NewClaudeModel(info)
	case Gemini:
		return NewGeminiModel(info)
	default:
		return nil
	}
}
