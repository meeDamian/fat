# fat

CLI for multi-LLM agent collaborative question answering: Models iteratively refine answers, exchange targeted suggestions, and rank final results.

**Note**: This is vibecoded and very much a work-in-progress (WIP). Expect bugs, incomplete features, and potential API changes.

## Setup

1. Ensure Go is installed.

2. Clone or download the project.

3. Install dependencies:
   ```
   go mod tidy
   ```

4. Build the binary (optional):
   ```
   go build -o fat cmd/fat/main.go
   ```

5. Set up API keys (choose one method):
   - Environment variables: `GROK_KEY`, `GPT_KEY`, `CLAUDE_KEY`, `GEMINI_KEY`
   - `.env` file with the same variables
   - `keys.json` with `{"grok": "key", "gpt": "key", "claude": "key", "gemini": "key"}`

## Usage

```
go run cmd/fat/main.go [flags] "your question here"
```

Or if built: `./fat [flags] "your question here"`

### Flags

- `--rounds=int`: Number of rounds (1-10, -1=auto estimate; default -1)
- `--full-context`: Use full history (default false)
- `--verbose`: Verbose output (default false)
- `--budget`: Estimate and confirm cost before running (default false)
- `--grok`: Include Grok model
- `--gpt`: Include GPT model
- `--claude`: Include Claude model
- `--gemini`: Include Gemini model
- `--no-grok`: Exclude Grok model
- `--no-gpt`: Exclude GPT model
- `--no-claude`: Exclude Claude model
- `--no-gemini`: Exclude Gemini model

### Examples

Single model:
```
go run cmd/fat/main.go --grok "What is the capital of France?"
```

Multi-model with auto rounds:
```
go run cmd/fat/main.go --verbose --grok --gpt "How to bake a cake?"
```

Exclude models:
```
go run cmd/fat/main.go --no-claude "What is AI?"
```

## How It Works

- **Single Model**: Direct answer from the selected model.
- **Multi-Model**: Iterative collaboration across rounds.
  - Round 1: Initial answers from all models.
  - Rounds 2-N: Each model refines its answer, critically analyzing gaps, incorporating suggestions targeted to it from other models, and providing new suggestions for others.
  - Final Round: Final refined answers.
  - Ranking: Models rank all final answers; the winner with the most top votes is selected.

Responses are parsed for structured format (answer + suggestions) or JSON. Suggestions are filtered and passed to targeted models in subsequent rounds.

## Models

- grok: grok-4-fast (xAI)
- gpt: gpt-5-mini (OpenAI)
- claude: claude-3.5-haiku (Anthropic)
- gemini: gemini-2.5-flash (Google)

## Notes

- At least one model must be selected for multi-model runs.
- Uses official SDKs for OpenAI, Anthropic, Gemini; direct HTTP for Grok.
- Costs estimated using provider rates; thinking tokens counted as output.
- Logs saved in `answers/` with timestamps and model details.
- Rates auto-fetched from providers if not cached or outdated.
