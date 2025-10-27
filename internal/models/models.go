package models

import (
	"github.com/meedamian/fat/internal/types"
)

// AllModels defines all available models with their configurations
// To change which models are active, modify this map
var AllModels = map[string]*types.ModelInfo{
	// Grok models
	Grok: {ID: Grok, Name: Grok4Fast, MaxTok: 131072, BaseURL: "https://api.x.ai/v1/chat/completions"},

	// OpenAI models
	GPT: {ID: GPT, Name: GPT5Mini, MaxTok: 16384, BaseURL: "https://api.openai.com/v1/chat/completions"},

	// Claude models - using 3.5 Haiku
	Claude: {ID: Claude, Name: Claude35Haiku, MaxTok: 200000, BaseURL: "https://api.anthropic.com/v1/messages"},

	// Gemini models
	Gemini: {ID: Gemini, Name: Gemini25Flash, MaxTok: 128000, BaseURL: "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent"},
}

// Alternative configurations (commented out - uncomment to use):
//
// For faster/cheaper models:
// Claude: {ID: Claude, Name: Claude35Haiku, MaxTok: 200000, BaseURL: "..."},
// GPT:    {ID: GPT, Name: GPT5Nano, MaxTok: 16384, BaseURL: "..."},
//
// For maximum capability:
// Claude: {ID: Claude, Name: Claude45Sonnet, MaxTok: 200000, BaseURL: "..."},
// GPT:    {ID: GPT, Name: GPT5, MaxTok: 16384, BaseURL: "..."},

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
