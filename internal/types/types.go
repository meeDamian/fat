package types

import (
	"context"
	"log/slog"
	"time"
)

// Rate holds pricing information with timestamp
type Rate struct {
	TS  int64   `json:"ts"`
	In  float64 `json:"in"`  // input cost per token
	Out float64 `json:"out"` // output cost per token
}

// ModelVariant contains properties specific to a model variant
// The variant name (API model name like "grok-4-fast") is the map key
type ModelVariant struct {
	MaxTok int64 // Max tokens for this variant
	Rate   Rate  // Pricing for this variant
}

// ModelFamily contains common properties for a model family
type ModelFamily struct {
	ID       string                  // Family ID (e.g., "grok", "gpt")
	Provider string                  // Provider name (e.g., "xAI", "OpenAI")
	BaseURL  string                  // API endpoint
	Variants map[string]ModelVariant // Available model variants
}

// ModelInfo contains model configuration (runtime instance)
type ModelInfo struct {
	ID             string
	Name           string
	MaxTok         int64
	BaseURL        string
	APIKey         string
	Client         any
	Logger         *slog.Logger
	RequestTimeout time.Duration
}

// DiscussionMessage represents a single message in a conversation thread
type DiscussionMessage struct {
	From    string // Model ID of sender
	Message string
	Round   int
}

// Reply represents a model's response
type Reply struct {
	Answer     string
	Rationale  string
	Discussion map[string]string // Agent -> Message to be added to discussion
	RawContent string            // For logging/debugging
}

// ModelResult holds the result of a model prompt
type ModelResult struct {
	Reply  Reply
	TokIn  int64
	TokOut int64
	Prompt string // For logging
}

// Meta contains metadata for prompt generation
type Meta struct {
	Round       int
	TotalRounds int
	OtherAgents []string // Agent count = len(OtherAgents) + 1
}

// Model interface for all AI providers
type Model interface {
	Prompt(ctx context.Context, question string, meta Meta, replies map[string]Reply, discussion map[string]map[string][]DiscussionMessage) (ModelResult, error)
}
