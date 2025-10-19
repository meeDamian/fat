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

// ModelInfo contains model configuration
type ModelInfo struct {
	ID      string
	Name    string
	MaxTok  int64
	BaseURL string
	APIKey  string
	Rates   Rate
	Client  any
	Logger  *slog.Logger
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
