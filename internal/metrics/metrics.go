package metrics

import (
	"sync"
	"time"
)

// RequestMetrics tracks metrics for a single request
type RequestMetrics struct {
	RequestID    string
	Question     string
	StartTime    time.Time
	EndTime      time.Time
	NumRounds    int
	NumModels    int
	ModelMetrics map[string]*ModelMetrics
	Winner       string
	mu           sync.RWMutex
}

// ModelMetrics tracks metrics for a single model
type ModelMetrics struct {
	ModelID       string
	RoundMetrics  []*RoundMetrics
	RankingTime   time.Duration
	RankingTokens TokenCount
	TotalTokens   TokenCount
	Errors        []string
	mu            sync.Mutex
}

// RoundMetrics tracks metrics for a single round
type RoundMetrics struct {
	Round     int
	StartTime time.Time
	Duration  time.Duration
	Tokens    TokenCount
	Error     string
}

// TokenCount tracks input and output tokens
type TokenCount struct {
	Input  int64
	Output int64
}

// NewRequestMetrics creates a new request metrics tracker
func NewRequestMetrics(requestID, question string, numRounds, numModels int) *RequestMetrics {
	return &RequestMetrics{
		RequestID:    requestID,
		Question:     question,
		StartTime:    time.Now(),
		NumRounds:    numRounds,
		NumModels:    numModels,
		ModelMetrics: make(map[string]*ModelMetrics),
	}
}

// AddModelMetrics initializes metrics for a model
func (rm *RequestMetrics) AddModelMetrics(modelID string) *ModelMetrics {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	mm := &ModelMetrics{
		ModelID:      modelID,
		RoundMetrics: make([]*RoundMetrics, 0),
		Errors:       make([]string, 0),
	}
	rm.ModelMetrics[modelID] = mm
	return mm
}

// RecordRound records metrics for a round
func (mm *ModelMetrics) RecordRound(round int, duration time.Duration, tokIn, tokOut int64, err error) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	roundMetric := &RoundMetrics{
		Round:     round,
		StartTime: time.Now().Add(-duration),
		Duration:  duration,
		Tokens: TokenCount{
			Input:  tokIn,
			Output: tokOut,
		},
	}

	if err != nil {
		roundMetric.Error = err.Error()
		mm.Errors = append(mm.Errors, err.Error())
	}

	mm.RoundMetrics = append(mm.RoundMetrics, roundMetric)
	mm.TotalTokens.Input += tokIn
	mm.TotalTokens.Output += tokOut
}

// RecordRanking records ranking metrics
func (mm *ModelMetrics) RecordRanking(duration time.Duration, tokIn, tokOut int64) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.RankingTime = duration
	mm.RankingTokens = TokenCount{
		Input:  tokIn,
		Output: tokOut,
	}
	mm.TotalTokens.Input += tokIn
	mm.TotalTokens.Output += tokOut
}

// Complete marks the request as complete
func (rm *RequestMetrics) Complete(winner string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.EndTime = time.Now()
	rm.Winner = winner
}

// Duration returns the total request duration
func (rm *RequestMetrics) Duration() time.Duration {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	if rm.EndTime.IsZero() {
		return time.Since(rm.StartTime)
	}
	return rm.EndTime.Sub(rm.StartTime)
}

// Summary returns a summary map for logging
func (rm *RequestMetrics) Summary() map[string]any {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	totalTokensIn := int64(0)
	totalTokensOut := int64(0)
	errorCount := 0

	for _, mm := range rm.ModelMetrics {
		mm.mu.Lock()
		totalTokensIn += mm.TotalTokens.Input
		totalTokensOut += mm.TotalTokens.Output
		errorCount += len(mm.Errors)
		mm.mu.Unlock()
	}

	return map[string]any{
		"request_id":       rm.RequestID,
		"duration_ms":      rm.Duration().Milliseconds(),
		"num_rounds":       rm.NumRounds,
		"num_models":       rm.NumModels,
		"total_tokens_in":  totalTokensIn,
		"total_tokens_out": totalTokensOut,
		"error_count":      errorCount,
		"winner":           rm.Winner,
	}
}
