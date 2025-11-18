# FAT - Multi-Agent AI Collaboration System

Web-based multi-LLM agent collaborative question answering: Models iteratively refine answers through structured discussion, exchange targeted feedback, and collectively rank final results.

## Features

- **Multi-Round Collaboration**: Models refine answers across multiple rounds (3-10 rounds configurable)
- **Structured Discussion**: Agents provide targeted feedback to specific peers in markdown format
- **Democratic Ranking**: All models vote on final answers using Borda count with tie handling
- **Medal System**: Gold (🏆), Silver (🥈), and Bronze (🥉) awards, with support for ties
- **Real-time WebSocket UI**: Live updates as models collaborate with responsive layout
- **Static HTML Export**: Self-contained snapshots of completed debates with all discussions
- **Model Flexibility**: Switch between variants per family via UI dropdowns
- **Structured Logging**: JSON-formatted logs with configurable levels
- **Configurable Timeouts**: Per-model request timeouts with context propagation
- **Comprehensive Testing**: Unit tests for prompt formatting, parsing, and ranking logic

## Setup

1. **Prerequisites**: Go 1.21+ installed

2. **Clone and install dependencies**:
   ```bash
   git clone <repo>
   cd fat
   go mod tidy
   ```

3. **Configure API keys** (choose one method):
   - **Environment variables**: `GROK_KEY`, `GPT_KEY`, `CLAUDE_KEY`, `GEMINI_KEY`, `DEEPSEEK_KEY`, `MISTRAL_KEY`
   - **`.env` file**: Same variables as above
   - **`keys.json`**: `{"grok": "key", "gpt": "key", "claude": "key", "gemini": "key", "deepseek": "key", "mistral": "key"}`

4. **Optional configuration** (environment variables):
   - `FAT_SERVER_ADDR`: Server address (default `:4444`)
   - `FAT_MODEL_TIMEOUT`: Model request timeout (default `30s`)
   - `FAT_LOG_LEVEL`: Log level - `debug`, `info`, `warn`, `error` (default `info`)

## Usage

### Start the server:
```bash
go run ./cmd/fat
```

The server will start on `http://localhost:4444` (or your configured address).

### Web Interface

Open `http://localhost:4444` in your browser and:
1. Enter your question
2. Select number of rounds (3-10, default 3) using the slider
3. Optionally switch model variants using the dropdowns on each card
4. Click "Launch Discussion" to start the collaboration
5. Watch real-time updates as models discuss and refine answers
6. See gold/silver/bronze medals awarded by democratic vote
7. Review agent discussions after ranking completes
8. Static HTML snapshot automatically saved to `answers/{timestamp}/`

### Run Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/shared/...

# With coverage
go test -cover ./...
```

## How It Works

### Multi-Round Collaboration

1. **Round 1**: Each model provides initial answer to the question
2. **Rounds 2-N**: Models:
   - Review answers from other agents
   - Read discussion messages directed at them
   - Refine their answer incorporating feedback
   - Provide new targeted suggestions to specific agents
3. **Ranking Phase**: All models independently rank all final answers
4. **Winner Selection**: Borda count aggregation determines the best answer

### Response Format

Models must respond in strict markdown format:

```markdown
# ANSWER
[Refined answer incorporating discussion, <300 words]

# RATIONALE
[Optional: Brief reasoning for changes]

# DISCUSSION
## With [AgentName]
[1-2 concise messages for that specific agent]
```

### Ranking System

- Each model ranks all agents (including itself) from best to worst using anonymized letters
- Borda count scoring: 1st place = n points, 2nd = n-1, etc.
- Models with same score tie and receive the same medal
- Gold (🏆), Silver (🥈), and Bronze (🥉) medals awarded to top 3 score tiers
- Multiple models can share the same medal level

## Architecture

```
cmd/fat/main.go           - Entry point
internal/
  config/                 - Configuration loading and logger setup
  db/                     - SQLite database for conversation history
  htmlexport/             - Static HTML snapshot generation
  metrics/                - Request metrics and cost tracking
  models/                 - Model family definitions and implementations
  orchestrator/           - Multi-round collaboration orchestration
  ranking/                - Model ranking and aggregation
  server/                 - HTTP server, WebSocket handler, API endpoints
  shared/                 - Prompt formatting, response parsing
  types/                  - Core types and interfaces
  utils/                  - Logging utilities
static/                   - Web UI assets (HTML, CSS, JavaScript)
answers/                  - Saved conversations and static exports
```

## Default Models

See `internal/models/models.go` - `DefaultModels` map:

- **Grok**: `grok-4-fast` (xAI) - 2M context, $0.20/$0.50 per 1M tokens
- **GPT**: `gpt-5-mini` (OpenAI) - 400K context, $0.25/$2.00 per 1M tokens
- **Claude**: `claude-4.5-haiku` (Anthropic) - 200K context, $1.00/$5.00 per 1M tokens
- **Gemini**: `gemini-2.5-pro` (Google) - 1M context, $1.25/$10.00 per 1M tokens
- **DeepSeek**: `deepseek-chat` (DeepSeek) - 128K context, $0.28/$0.42 per 1M tokens
- **Mistral**: `mistral-medium` (Mistral AI) - 128K context, $0.40/$2.00 per 1M tokens

All models can be switched via UI dropdowns or by changing `DefaultModels` in code.

## Logging

All conversations are logged to `answers/` directory:
- Format: `{timestamp}_{sequence}_{round}_{model}.log`
- Contains both prompt and raw response
- Structured JSON logs to stdout for server events

## Development

### Adding a New Model Family

1. **Add constants** in `internal/models/constants.go` for family ID and variant names
2. **Add to `ModelFamilies`** in `internal/models/models.go`:
   ```go
   NewFamily: {
       ID:       NewFamily,
       Provider: "Provider Name",
       BaseURL:  "https://api.provider.com/endpoint",
       Variants: map[string]types.ModelVariant{
           NewModelVariant: {MaxTok: 128_000, Rate: types.Rate{In: 1.0, Out: 2.0}},
       },
   },
   ```
3. **Add to `DefaultModels`** to set the default variant
4. **Create implementation** in `internal/models/newfamily.go` implementing `types.Model` interface
5. **Add case** to `NewModel()` factory function
6. **Configure API key** loading in `internal/apikeys/apikeys.go`

See `MODELS.md` and `UI_MODEL_SELECTION.md` for detailed documentation.

### Key Interfaces

```go
type Model interface {
    Prompt(ctx context.Context, question string, meta Meta, 
           replies map[string]Reply, 
           discussion map[string]map[string][]DiscussionMessage) (ModelResult, error)
}

type ModelVariant struct {
    MaxTok int64 // Max tokens
    Rate   Rate  // Pricing
}
```

## Notes

- Uses official SDKs for OpenAI, Anthropic, Gemini; direct HTTP for Grok, DeepSeek, Mistral
- Context timeouts prevent hanging on slow providers
- Discussion tracking handles multi-agent conversations with proper pairing
- Markdown parsing uses goldmark for robust section extraction
- All models participate in ranking using anonymized agent letters
- Static HTML exports include full conversation history and styling
