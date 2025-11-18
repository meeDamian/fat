# Model Configuration Guide

## Quick Model Switching

The project uses a hierarchical model structure with **families** (xAI, OpenAI, Anthropic, Google) and **variants** (specific models).

### Single Place to Change Default Models

Edit `internal/models/models.go` - change the `DefaultModels` map:

```go
var DefaultModels = map[string]string{
    Grok:   Grok4Fast,
    GPT:    GPT5Mini,
    Claude: Claude35Haiku,
    Gemini: Gemini25Flash,
}
```

That's it! Everything else (API keys, logging, database) automatically works.

## Available Model Variants

### Grok (xAI)
[Models list](https://docs.x.ai/docs/models) | Defined in `grok.go`

- `Grok4Fast` - "grok-4-fast" (2M tokens)
- `Grok4FastNonReasoning` - "grok-4-fast-non-reasoning" (2M tokens)
- `GrokCodeFast1` - "grok-code-fast-1" (256K tokens)
- `Grok4` - "grok-4" (256K tokens)
- `Grok3Mini` - "grok-3-mini" (131K tokens)
- `Grok3` - "grok-3" (131K tokens)

### GPT (OpenAI)
[Models list](https://platform.openai.com/docs/models) | Defined in `openai.go`

- `GPT5Pro` - "gpt-5-pro" (400K tokens)
- `GPT5` - "gpt-5" (400K tokens)
- `GPT5Mini` - "gpt-5-mini" (400K tokens)
- `GPT5Nano` - "gpt-5-nano" (400K tokens)
- `GPT5Codex` - "gpt-5-codex" (400K tokens)
- `GPT41` - "gpt-4.1" (1M tokens)
- `GPT41Mini` - "gpt-4.1-mini" (1M tokens)
- `GPT41Nano` - "gpt-4.1-nano" (1M tokens)

### Claude (Anthropic)
[Models list](https://docs.claude.com/en/docs/about-claude/models/overview) | Defined in `claude.go`

- `Claude45Sonnet` - "claude-sonnet-4-5" (200K tokens) ⭐ **Latest**
- `Claude45Haiku` - "claude-haiku-4-5" (200K tokens)
- `Claude41Opus` - "claude-opus-4-1" (200K tokens)
- `Claude4Sonnet` - "claude-sonnet-4-0" (200K tokens)
- `Claude37Sonnet` - "claude-3-7-sonnet-latest" (200K tokens)
- `Claude4Opus` - "claude-opus-4-0" (200K tokens)
- `Claude35Haiku` - "claude-3-5-haiku-latest" (200K tokens) 💨 **Fastest**

**Note**: Claude Sonnet 4.5 and 4 support 1M token context with `context-1m-2025-08-07` beta header. Long context pricing applies beyond 200K tokens.

### Gemini (Google)
[Models list](https://ai.google.dev/gemini-api/docs/models) | Defined in `gemini.go`

- `Gemini25Pro` - "gemini-2.5-pro" (1M tokens)
- `Gemini25Flash` - "gemini-2.5-flash" (1M tokens)
- `Gemini25FlashLite` - "gemini-2.5-flash-lite" (1M tokens)
- `Gemini20Flash` - "gemini-2.0-flash" (1M tokens)
- `Gemini20FlashLite` - "gemini-2.0-flash-lite" (1M tokens)

## Configuration Examples

### Maximum Capability
```go
var DefaultModels = map[string]string{
    Grok:   Grok4,           // Most capable Grok
    GPT:    GPT5Pro,         // Most capable GPT
    Claude: Claude45Sonnet,  // Latest Claude
    Gemini: Gemini25Pro,     // Most capable Gemini
}
```

### Speed/Cost Optimization
```go
var DefaultModels = map[string]string{
    Grok:   Grok3Mini,       // Fastest Grok
    GPT:    GPT5Nano,        // Smallest GPT
    Claude: Claude35Haiku,   // Fastest Claude
    Gemini: Gemini20Flash,   // Fastest Gemini
}
```

### Code-Focused
```go
var DefaultModels = map[string]string{
    Grok:   GrokCodeFast1,   // Code-specialized
    GPT:    GPT5Codex,       // Code-specialized
    Claude: Claude45Sonnet,  // Strong at code
    Gemini: Gemini25Pro,     // Strong at code
}
```

### Disable a Model Family
Remove it from `DefaultModels`:
```go
var DefaultModels = map[string]string{
    // Grok:   Grok4Fast,    // Disabled
    GPT:    GPT5Mini,
    Claude: Claude35Haiku,
    Gemini: Gemini25Flash,
}
```

## How It Works

### ModelFamilies
Defines all available models with common properties per provider:

```go
ModelFamilies = map[string]types.ModelFamily{
    Grok: {
        ID:       "grok",
        Provider: "xAI",
        BaseURL:  "https://api.x.ai/v1/chat/completions",
        Variants: map[string]types.ModelVariant{
            Grok4Fast: {Name: "grok-4-fast", MaxTok: 2_000_000},
            Grok4:     {Name: "grok-4", MaxTok: 256_000},
            // ... all variants
        },
    },
}
```

### DefaultModels
Single source of truth - change here to switch default models:

```go
var DefaultModels = map[string]string{
    Grok:   Grok4Fast,      // Family → Variant
    GPT:    GPT5Mini,
    Claude: Claude35Haiku,
    Gemini: Gemini25Flash,
}
```

### Runtime
`buildDefaultModels()` constructs `ModelInfo` instances from families + default models. API keys are loaded by family ID (not model name), so switching variants requires no other changes.

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

## API Keys

Keys are loaded by **family ID**, not model name. Set one of:

1. **Environment variables**:
   ```bash
   export GROK_KEY="xai-..."
   export GPT_KEY="sk-..."
   export CLAUDE_KEY="sk-ant-..."
   export GEMINI_KEY="AI..."
   ```

2. **`.env` file**:
   ```
   GROK_KEY=xai-...
   GPT_KEY=sk-...
   CLAUDE_KEY=sk-ant-...
   GEMINI_KEY=AI...
   ```

3. **`keys.json` file**:
   ```json
   {
     "grok": "xai-...",
     "gpt": "sk-...",
     "claude": "sk-ant-...",
     "gemini": "AI..."
   }
   ```

## Troubleshooting

### Timeout Errors
If you see `context deadline exceeded`:

1. Increase timeout: `export FAT_MODEL_TIMEOUT=90s`
2. Use faster models (e.g., `Claude35Haiku` instead of `Claude45Sonnet`)
3. Reduce number of rounds
4. Check network connectivity

### Adding New Models

1. Add constant to model file (e.g., `claude.go`):
   ```go
   ClaudeNewModel = "claude-new-model"
   ```

2. Add to `ModelFamilies` variants in `models.go`:
   ```go
   Claude: {
       Variants: map[string]types.ModelVariant{
           ClaudeNewModel: {Name: ClaudeNewModel, MaxTok: 200_000},
           // ... existing variants
       },
   }
   ```

3. Use it in `DefaultModels`:
   ```go
   Claude: ClaudeNewModel,
   ```

No other changes needed!
