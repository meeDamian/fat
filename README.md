# fat

CLI for multi-LLM agent swarm on a question: Optional round estimation, iterate refinements/suggestions, rank finals.

## Setup

1. Initialize the module:
   ```
   go mod init github.com/meedamian/fat
   go mod tidy
   ```

2. Build the binary:
   ```
   go build -o fat cmd/fat/main.go
   ```

3. Set up API keys:
   - Environment variables: `GROK_4_FAST`, `GPT_5_MINI`, `CLAUDE_3_5_HAIKU`, `GEMINI_2_5_FLASH`
   - Or create `.env` file with the same variables
   - Or create `keys.json` with `{"grok-4-fast": "key", ...}`

## Usage

```
./fat [flags] question text here
```

### Flags

- `--rounds=int`: Number of rounds (1-10, -1=auto; default -1)
- `--full-context`: Use full history (default false)
- `--verbose`: Verbose output (default false)
- `--budget`: Estimate and confirm budget (default false)
- `--model`: Include model (A/B/C/D), can specify multiple
- `--no-model`: Exclude model (A/B/C/D), can specify multiple

### Examples

Single model:
```
./fat --model A What is the capital of France?
```

Multi model auto rounds:
```
./fat --verbose How to bake a cake?
```

## Models

- A: grok-4-fast (xAI)
- B: gpt-5-mini (OpenAI)
- C: claude-3.5-haiku (Anthropic)
- D: gemini-2.5-flash (Google)

## Notes

- At least one model must be selected.
- Uses SDKs for OpenAI, Anthropic, Gemini; HTTP for Grok.
- Costs include thinking tokens as output cost.
- Logs per model in `answers/` with Unix timestamps.
- Rates auto-fetched weekly from provider sites.
