# Model Configuration Guide

## Quick Model Switching

All model names are defined as constants in their respective files in `internal/models/`. To change which models are used:

### 1. Edit `internal/models/models.go`

Change the `AllModels` map entries to use different model variants:

```go
var AllModels = map[string]*types.ModelInfo{
    Grok:   {ID: Grok, Name: Grok4Fast, MaxTok: 131072, BaseURL: "..."},
    GPT:    {ID: GPT, Name: GPT5Mini, MaxTok: 16384, BaseURL: "..."},
    Claude: {ID: Claude, Name: Claude45Sonnet, MaxTok: 200000, BaseURL: "..."},
    Gemini: {ID: Gemini, Name: Gemini25Flash, MaxTok: 128000, BaseURL: "..."},
}
```

**Current default**: Claude 4.5 Sonnet (most capable Claude model)

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
- `Claude45Sonnet` - "claude-4-5-sonnet-20250514" ⭐ **Latest & most capable**
- `Claude35Haiku` - "claude-3-5-haiku-20241022" (fastest, cheapest)
- `Claude35Sonnet` - "claude-3-5-sonnet-20241022"
- `Claude3Opus` - "claude-3-opus-20240229"

**Gemini** (defined in `gemini.go`):
- `Gemini25Flash` - "gemini-2.5-flash"
- `Gemini20Flash` - "gemini-2.0-flash"
- `Gemini15Pro` - "gemini-1.5-pro"

### 3. Example Configurations

**For maximum capability** (current default):
```go
var AllModels = map[string]*types.ModelInfo{
    Grok:   {ID: Grok, Name: Grok4FastReasoning, MaxTok: 131072, BaseURL: "..."},
    GPT:    {ID: GPT, Name: GPT5, MaxTok: 16384, BaseURL: "..."},
    Claude: {ID: Claude, Name: Claude45Sonnet, MaxTok: 200000, BaseURL: "..."},
    Gemini: {ID: Gemini, Name: Gemini20Flash, MaxTok: 128000, BaseURL: "..."},
}
```

**For speed/cost optimization**:
```go
var AllModels = map[string]*types.ModelInfo{
    Grok:   {ID: Grok, Name: Grok4Fast, MaxTok: 131072, BaseURL: "..."},
    GPT:    {ID: GPT, Name: GPT5Nano, MaxTok: 16384, BaseURL: "..."},
    Claude: {ID: Claude, Name: Claude35Haiku, MaxTok: 200000, BaseURL: "..."},
    Gemini: {ID: Gemini, Name: Gemini25Flash, MaxTok: 128000, BaseURL: "..."},
}
```

**To run with only 2 models** (comment out the others):
```go
var AllModels = map[string]*types.ModelInfo{
    // Grok:   {ID: Grok, Name: Grok4Fast, MaxTok: 131072, BaseURL: "..."},
    GPT:    {ID: GPT, Name: GPT5Mini, MaxTok: 16384, BaseURL: "..."},
    Claude: {ID: Claude, Name: Claude45Sonnet, MaxTok: 200000, BaseURL: "..."},
    // Gemini: {ID: Gemini, Name: Gemini25Flash, MaxTok: 128000, BaseURL: "..."},
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
