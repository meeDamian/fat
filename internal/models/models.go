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
	// Models list: https://docs.x.ai/docs/models
	Grok: {
		ID:       Grok,
		Provider: "xAI",
		BaseURL:  "https://api.x.ai/v1/chat/completions",
		Variants: map[string]types.ModelVariant{
			Grok4Fast:             {Name: Grok4Fast, MaxTok: 2_000_000, Rate: types.Rate{In: 0.2, Out: 0.5}},
			Grok4FastNonReasoning: {Name: Grok4FastNonReasoning, MaxTok: 2_000_000, Rate: types.Rate{In: 0.2, Out: 0.5}},
			GrokCodeFast1:         {Name: GrokCodeFast1, MaxTok: 256_000, Rate: types.Rate{In: 0.2, Out: 1.5}},
			Grok4:                 {Name: Grok4, MaxTok: 256_000, Rate: types.Rate{In: 3.0, Out: 15.0}},
			Grok3Mini:             {Name: Grok3Mini, MaxTok: 131_072, Rate: types.Rate{In: 0.3, Out: 0.5}},
			Grok3:                 {Name: Grok3, MaxTok: 131_072, Rate: types.Rate{In: 3.0, Out: 15.0}},
		},
	},

	// Models list: https://platform.openai.com/docs/models
	GPT: {
		ID:       GPT,
		Provider: "OpenAI",
		BaseURL:  "https://api.openai.com/v1/chat/completions",
		Variants: map[string]types.ModelVariant{
			GPT5Pro:   {Name: GPT5Pro, MaxTok: 400_000, Rate: types.Rate{In: 15.0, Out: 120.0}},
			GPT5:      {Name: GPT5, MaxTok: 400_000, Rate: types.Rate{In: 1.25, Out: 10.0}},
			GPT5Mini:  {Name: GPT5Mini, MaxTok: 400_000, Rate: types.Rate{In: 0.25, Out: 2.0}},
			GPT5Nano:  {Name: GPT5Nano, MaxTok: 400_000, Rate: types.Rate{In: 0.05, Out: 0.4}},
			GPT5Codex: {Name: GPT5Codex, MaxTok: 400_000, Rate: types.Rate{In: 1.25, Out: 10.0}},
			GPT41:     {Name: GPT41, MaxTok: 1_047_576, Rate: types.Rate{In: 2.0, Out: 8.0}},
			GPT41Mini: {Name: GPT41Mini, MaxTok: 1_047_576, Rate: types.Rate{In: 0.4, Out: 1.6}},
			GPT41Nano: {Name: GPT41Nano, MaxTok: 1_047_576, Rate: types.Rate{In: 0.1, Out: 0.4}},
		},
	},

	// Models list: https://docs.claude.com/en/docs/about-claude/models/overview
	Claude: {
		ID:       Claude,
		Provider: "Anthropic",
		BaseURL:  "https://api.anthropic.com/v1/messages",
		Variants: map[string]types.ModelVariant{
			// NOTE: Claude Sonnet 4.5 supports a 1M token context window when using the context-1m-2025-08-07 beta header. Long context pricing applies to requests exceeding 200K tokens.
			// NOTE: Claude Sonnet 4 supports a 1M token context window when using the context-1m-2025-08-07 beta header. Long context pricing applies to requests exceeding 200K tokens.
			Claude45Sonnet: {Name: Claude45Sonnet, MaxTok: 200_000, Rate: types.Rate{In: 3.0, Out: 15.0}},
			Claude45Haiku:  {Name: Claude45Haiku, MaxTok: 200_000, Rate: types.Rate{In: 1.0, Out: 5.0}},
			Claude41Opus:   {Name: Claude41Opus, MaxTok: 200_000, Rate: types.Rate{In: 15.0, Out: 75.0}},
			Claude4Sonnet:  {Name: Claude4Sonnet, MaxTok: 200_000, Rate: types.Rate{In: 3.0, Out: 15.0}},
			Claude37Sonnet: {Name: Claude37Sonnet, MaxTok: 200_000, Rate: types.Rate{In: 3.0, Out: 15.0}},
			Claude4Opus:    {Name: Claude4Opus, MaxTok: 200_000, Rate: types.Rate{In: 15.0, Out: 75.0}},
			Claude35Haiku:  {Name: Claude35Haiku, MaxTok: 200_000, Rate: types.Rate{In: 0.8, Out: 4.0}},
		},
	},

	// Models list: https://ai.google.dev/gemini-api/docs/models
	Gemini: {
		ID:       Gemini,
		Provider: "Google",
		BaseURL:  "https://generativelanguage.googleapis.com/v1beta/models/{model}:generateContent", // Updated to placeholder for flexibility.
		Variants: map[string]types.ModelVariant{
			Gemini25Pro:       {Name: Gemini25Pro, MaxTok: 1_048_576, Rate: types.Rate{In: 1.25, Out: 10.0}},
			Gemini25Flash:     {Name: Gemini25Flash, MaxTok: 1_048_576, Rate: types.Rate{In: 0.30, Out: 2.5}},
			Gemini25FlashLite: {Name: Gemini25FlashLite, MaxTok: 1_048_576, Rate: types.Rate{In: 0.1, Out: 0.4}},
			Gemini20Flash:     {Name: Gemini20Flash, MaxTok: 1_048_576, Rate: types.Rate{In: 0.1, Out: 0.4}},
			Gemini20FlashLite: {Name: Gemini20FlashLite, MaxTok: 1_048_576, Rate: types.Rate{In: 0.075, Out: 0.3}},
		},
	},

	// Models list: https://api-docs.deepseek.com/
	DeepSeek: {
		ID:       DeepSeek,
		Provider: "DeepSeek",
		BaseURL:  "https://api.deepseek.com/v1",
		Variants: map[string]types.ModelVariant{
			DeepSeekChat:  {Name: DeepSeekChat, MaxTok: 128_000, Rate: types.Rate{In: 0.28, Out: 0.42}},
			DeepSeekCoder: {Name: DeepSeekCoder, MaxTok: 128_000, Rate: types.Rate{In: 0.28, Out: 0.42}},
		},
	},
}

// ActiveModels defines which model variant to use for each family
// Change the variant name here to switch models
var ActiveModels = map[string]string{
	Grok:     Grok4Fast,
	GPT:      GPT5Mini,
	Claude:   Claude45Haiku,
	Gemini:   Gemini25Pro,
	DeepSeek: DeepSeekChat,
}

// AllModels builds runtime ModelInfo instances from families and active models
var AllModels = buildActiveModels()

// buildActiveModels constructs ModelInfo instances from ModelFamilies and ActiveModels
func buildActiveModels() map[string]*types.ModelInfo {
	models := make(map[string]*types.ModelInfo)

	for familyID, variantName := range ActiveModels {
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
			Name:    variant.Name,
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
	default:
		return nil
	}
}
