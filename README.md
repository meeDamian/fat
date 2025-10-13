# fat

CLI for multi-LLM agent swarm on a question: Optional round estimation, iterate refinements/suggestions, rank finals.

## Setup

1. Initialize the module:
   ```
   go mod init github.com/meedamian/fat
   go mod tidy
   ```

2. Run the application:
   ```
   go run cmd/fat/main.go [flags] "your question here"
   ```

   Or build the binary first:
   ```
   go build -o fat cmd/fat/main.go
   ./fat [flags] "your question here"
   ```

3. Set up API keys:
   - Environment variables: `GROK_KEY`, `GPT_KEY`, `CLAUDE_KEY`, `GEMINI_KEY`
   - Or create `.env` file with the same variables
   - Or create `keys.json` with `{"grok": "key", "gpt": "key", "claude": "key", "gemini": "key"}`

## Usage

```
go run cmd/fat/main.go [flags] question text here
```

Or if built: `./fat [flags] question text here`

### Flags

- `--rounds=int`: Number of rounds (1-10, -1=auto; default -1)
- `--full-context`: Use full history (default false)
- `--verbose`: Verbose output (default false)
- `--budget`: Estimate and confirm budget (default false)
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
go run cmd/fat/main.go --grok What is the capital of France?
```

Multi model auto rounds:
```
go run cmd/fat/main.go --verbose --grok --gpt How to bake a cake?
```

Exclude models:
```
go run cmd/fat/main.go --no-claude What is AI?
```

## Models

- grok: grok-4-fast (xAI)
- gpt: gpt-5-mini (OpenAI)
- claude: claude-3.5-haiku (Anthropic)
- gemini: gemini-2.5-flash (Google)

## Notes

- At least one model must be selected.
- Uses SDKs for OpenAI, Anthropic, Gemini; HTTP for Grok.
- Costs include thinking tokens as output cost.
- Logs per model in `answers/` with Unix timestamps.
- Rates auto-fetched weekly from provider sites.
