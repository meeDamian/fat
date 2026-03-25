package models

import (
	"fmt"

	"github.com/meedamian/fat/internal/types"
)

// ModelFamilies defines all available model families with their variants
//
// PRICING NOTES:
// - Rate.In = input cost per 1M tokens (e.g., 3.0 means $3.00 per 1M input tokens)
// - Rate.Out = output cost per 1M tokens (e.g., 15.0 means $15.00 per 1M output tokens)
// - Set to 0.0 if pricing is not available yet
// - Update pricing from provider documentation:
//   - Grok: https://docs.x.ai/docs/models
//   - GPT: https://openai.com/api/pricing/
//   - Claude: https://www.anthropic.com/pricing
//   - Gemini: https://ai.google.dev/pricing
//   - DeepSeek: https://platform.deepseek.com/api-docs/pricing/
var ModelFamilies = map[string]types.ModelFamily{
	Grok:     GrokFamily,
	GPT:      GPTFamily,
	Claude:   ClaudeFamily,
	Gemini:   GeminiFamily,
	DeepSeek: DeepSeekFamily,
	Mistral:  MistralFamily,
}

// DefaultModels defines which model variant to use for each family by default
// Change the variant name here to switch default models
var DefaultModels = map[string]string{
	Grok:     Grok420MultiAgent,
	GPT:      GPT5Mini,
	Claude:   Claude46Opus,
	Gemini:   Gemini31FlashLite,
	DeepSeek: DeepSeekChat,
	Mistral:  MistralLarge,
}

// AllModels builds runtime ModelInfo instances from families and default models
var AllModels = buildDefaultModels()

// buildDefaultModels constructs ModelInfo instances from ModelFamilies and DefaultModels
func buildDefaultModels() map[string]*types.ModelInfo {
	models := make(map[string]*types.ModelInfo)

	for familyID, variantName := range DefaultModels {
		family, ok := ModelFamilies[familyID]
		if !ok {
			panic(fmt.Sprintf("Unknown model family: %s", familyID))
		}

		variant, ok := family.Variants[variantName]
		if !ok {
			panic(fmt.Sprintf("Unknown variant %s for family %s", variantName, familyID))
		}

		models[familyID] = &types.ModelInfo{
			ID:      family.ID,
			Name:    variantName,
			MaxTok:  variant.MaxTok,
			BaseURL: family.BaseURL,
		}
	}

	return models
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
	case DeepSeek:
		return NewDeepSeekModel(info)
	case Mistral:
		return NewMistralModel(info)
	default:
		return nil
	}
}
