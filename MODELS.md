# Model Configuration Guide

## Quick Model Switching

The project uses a hierarchical model structure with **families** (xAI, OpenAI, Anthropic, Google) and **variants** (specific models).

### Single Place to Change Models

Edit `internal/models/models.go` - change the `ActiveModels` map:

```go
var ActiveModels = map[string]string{
    Grok:   Grok4Fast,      // Change to: Grok4FastReasoning, Grok4FastNonReasoning, GrokCodeFast1
    GPT:    GPT5Mini,       // Change to: GPT5Nano, GPT5, etc.
    Claude: Claude35Haiku,  // Change to: Claude45Sonnet, Claude35Sonnet, Claude3Opus
    Gemini: Gemini25Flash,  // Change to: Gemini20Flash, Gemini25Pro
}
```

That's it! Everything else (API keys, logging, database) automatically works.

### 2. Available Model Constants

**Grok** (defined in `grok.go`):
- `Grok4Fast` - "grok-4-fast"
- `Grok4FastReasoning` - "grok-4-fast-reasoning"
- `Grok4FastNonReasoning` - "grok-4-fast-non-reasoning"
- `GrokCodeFast1` - "grok-code-fast-1"

**GPT** (defined in `openai.go`):
- `GPT5Nano` - "gpt-5-nano"
- `GPT5Mini` - "gpt-5-mini"
- `GPT5` - "gpt-5"

**Claude** (defined in `claude.go`):
- `Claude45Sonnet` - "claude-sonnet-4-5-20250929" ⭐ **Latest & most capable**
- `Claude35Haiku` - "claude-3-5-haiku-20241022" (fastest, cheapest)
- `Claude35Sonnet` - "claude-3-5-sonnet-20241022"
- `Claude3Opus` - "claude-3-opus-20240229"

**Gemini** (defined in `gemini.go`):
- `Gemini25Flash` - "gemini-2.5-flash"
- `Gemini20Flash` - "gemini-2.0-flash"
- `Gemini25Pro` - "gemini-2.5-pro"

## Model Structure

### ModelFamilies
Defines all available models with common properties:

```go
ModelFamilies = map[string]types.ModelFamily{
    Grok: {
        ID:       "grok",
        Provider: "xAI",
        BaseURL:  "https://api.x.ai/v1/chat/completions",
        Variants: map[string]types.ModelVariant{
            Grok4Fast: {Name: "grok-4-fast", MaxTok: 131072},
            // ... other variants
        },
    },
    // ... other families
}
```

### ActiveModels
Single source of truth for which variant to use:

```go
var ActiveModels = map[string]string{
    Grok:   Grok4Fast,
    GPT:    GPT5Mini,
    Claude: Claude35Haiku,
    Gemini: Gemini25Flash,
}
```

### Examples

**For maximum capability**:
```go
var ActiveModels = map[string]string{
    Grok:   Grok4FastReasoning,
    GPT:    GPT5,
    Claude: Claude45Sonnet,
    Gemini: Gemini25Pro,
}
```

**For speed/cost optimization**:
```go
var ActiveModels = map[string]string{
    Grok:   Grok4Fast,
    GPT:    GPT5Nano,
    Claude: Claude35Haiku,
    Gemini: Gemini25Flash,
}
```

**To disable a model family** (remove from ActiveModels):
```go
var ActiveModels = map[string]string{
    // Grok:   Grok4Fast,  // Disabled
    GPT:    GPT5Mini,
    Claude: Claude45Sonnet,
    Gemini: Gemini25Flash,
}
```

## Timeout Configuration

**Default timeout**: 60 seconds per model request

### Change via Environment Variable

```bash
export FAT_MODEL_TIMEOUT=90s  # 90 seconds
export FAT_MODEL_TIMEOUT=2m   # 2 minutes
```

### Change Default in Code

Edit `internal/config/config.go`:

```go
ModelRequestTimeout: 60 * time.Second,  // Change this value
```

## Common Timeout Issues

If you see errors like:
```
context deadline exceeded
context cancelled during backoff
```

**Solutions**:
1. Increase timeout: `export FAT_MODEL_TIMEOUT=90s`
2. Use faster models (e.g., `GPT5Mini` instead of `GPT5`)
3. Reduce number of rounds in the question
4. Check network connectivity

## Adding New Models

1. Add constant to appropriate file (`grok.go`, `openai.go`, etc.)
2. Update `ModelMap` in `models.go`
3. Ensure API key is set in environment or `.env` file
4. Update `MaxTok` if the model has different limits
