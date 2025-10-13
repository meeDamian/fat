package types

// Response represents a model's refined answer and suggestions
type Response struct {
	Refined     string            `json:"refined"`
	Suggestions map[string]string `json:"suggestions"`
}

// History maps model IDs to their response history
type History map[string][]Response

// Rate holds pricing information with timestamp
type Rate struct {
	TS  int64   `json:"ts"`
	In  float64 `json:"in"`  // input cost per token
	Out float64 `json:"out"` // output cost per token
}

// Rank maps model IDs to their ranking scores
type Rank map[string]int

// ModelInfo contains model configuration
type ModelInfo struct {
	ID      string
	Name    string
	MaxTok  int64
	BaseURL string
	APIKey  string
	Rates   Rate
	Client  any
}

// RoundRes holds the result of a round call
type RoundRes struct {
	ID   string
	Resp Response
	Err  error
}
