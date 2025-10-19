# FAT - Multi-Agent AI Collaboration System

Web-based multi-LLM agent collaborative question answering: Models iteratively refine answers through structured discussion, exchange targeted feedback, and collectively rank final results.

## Features

- **Multi-Round Collaboration**: Models refine answers across multiple rounds, incorporating feedback from peers
- **Structured Discussion**: Agents provide targeted suggestions to specific peers using markdown format
- **Democratic Ranking**: All models vote on final answers using Borda count aggregation
- **Real-time WebSocket UI**: Live updates as models collaborate
- **Structured Logging**: JSON-formatted logs with configurable levels
- **Configurable Timeouts**: Per-model request timeouts with context propagation
- **Comprehensive Testing**: Unit tests for prompt formatting and response parsing

## Setup

1. **Prerequisites**: Go 1.21+ installed

2. **Clone and install dependencies**:
   ```bash
   git clone <repo>
   cd fat
   go mod tidy
   ```

3. **Configure API keys** (choose one method):
   - **Environment variables**: `GROK_KEY`, `GPT_KEY`, `CLAUDE_KEY`, `GEMINI_KEY`
   - **`.env` file**: Same variables as above
   - **`keys.json`**: `{"grok": "key", "gpt": "key", "claude": "key", "gemini": "key"}`

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
2. Select number of rounds (3-10, default 3)
3. Click "Ask" to start the collaboration
4. Watch real-time updates as models discuss and refine answers
5. See the final winner selected by democratic vote

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

- Each model ranks all agents (including itself) from best to worst
- Borda count scoring: 1st place = n points, 2nd = n-1, etc.
- Highest total score wins
- Ties broken by first responder

## Architecture

```
cmd/fat/main.go           - HTTP server, WebSocket handler, orchestration
internal/
  config/                 - Configuration loading and logger setup
  models/                 - Model implementations (Grok, GPT, Claude, Gemini)
  shared/                 - Prompt formatting, response parsing, ranking
  types/                  - Core types and interfaces
  utils/                  - Logging utilities
static/                   - Web UI assets
```

## Models

- **Grok**: `grok-4-fast` (xAI) - 131K context
- **GPT**: `gpt-5-mini` (OpenAI) - 16K context  
- **Claude**: `claude-3.5-haiku` (Anthropic) - 200K context
- **Gemini**: `gemini-2.5-flash` (Google) - 128K context

## Logging

All conversations are logged to `answers/` directory:
- Format: `{timestamp}_{sequence}_{round}_{model}.log`
- Contains both prompt and raw response
- Structured JSON logs to stdout for server events

## Development

### Adding a New Model

1. Create `internal/models/newmodel.go`
2. Implement `types.Model` interface
3. Add to `models.ModelMap` in `internal/models/models.go`
4. Add case to `NewModel()` factory function
5. Configure API key loading in `cmd/fat/main.go`

### Key Interfaces

```go
type Model interface {
    Prompt(ctx context.Context, question string, meta Meta, 
           replies map[string]string, discussion map[string][]string) (ModelResult, error)
}

type Meta struct {
    Round       int
    TotalRounds int
    OtherAgents []string
}
```

## Notes

- Uses official SDKs for OpenAI, Anthropic, Gemini; direct HTTP for Grok
- Context timeouts prevent hanging on slow providers
- Discussion tracking properly handles multi-agent conversations
- Markdown parsing uses goldmark for robust section extraction
- All models participate in ranking for democratic selection
