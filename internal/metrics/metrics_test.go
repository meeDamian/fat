package metrics

import (
	"errors"
	"testing"
	"time"
)

func TestNewRequestMetrics(t *testing.T) {
	rm := NewRequestMetrics("test-123", "What is AI?", 3, 4)

	if rm.RequestID != "test-123" {
		t.Errorf("Expected RequestID 'test-123', got %s", rm.RequestID)
	}

	if rm.Question != "What is AI?" {
		t.Errorf("Expected question 'What is AI?', got %s", rm.Question)
	}

	if rm.NumRounds != 3 {
		t.Errorf("Expected 3 rounds, got %d", rm.NumRounds)
	}

	if rm.NumModels != 4 {
		t.Errorf("Expected 4 models, got %d", rm.NumModels)
	}

	if rm.ModelMetrics == nil {
		t.Error("ModelMetrics map is nil")
	}
}

func TestAddModelMetrics(t *testing.T) {
	rm := NewRequestMetrics("test-123", "Test", 1, 1)

	mm := rm.AddModelMetrics("grok")

	if mm == nil {
		t.Fatal("AddModelMetrics returned nil")
	}

	if mm.ModelID != "grok" {
		t.Errorf("Expected ModelID 'grok', got %s", mm.ModelID)
	}

	if rm.ModelMetrics["grok"] == nil {
		t.Error("Model metrics not added to map")
	}
}

func TestRecordRound(t *testing.T) {
	mm := &ModelMetrics{
		ModelID:      "grok",
		RoundMetrics: make([]*RoundMetrics, 0),
		Errors:       make([]string, 0),
	}

	mm.RecordRound(1, 1*time.Second, 100, 50, nil)

	if len(mm.RoundMetrics) != 1 {
		t.Fatalf("Expected 1 round metric, got %d", len(mm.RoundMetrics))
	}

	rm := mm.RoundMetrics[0]
	if rm.Round != 1 {
		t.Errorf("Expected round 1, got %d", rm.Round)
	}

	if rm.Duration != 1*time.Second {
		t.Errorf("Expected duration 1s, got %v", rm.Duration)
	}

	if rm.Tokens.Input != 100 {
		t.Errorf("Expected 100 input tokens, got %d", rm.Tokens.Input)
	}

	if rm.Tokens.Output != 50 {
		t.Errorf("Expected 50 output tokens, got %d", rm.Tokens.Output)
	}

	if mm.TotalTokens.Input != 100 {
		t.Errorf("Expected total input 100, got %d", mm.TotalTokens.Input)
	}

	if mm.TotalTokens.Output != 50 {
		t.Errorf("Expected total output 50, got %d", mm.TotalTokens.Output)
	}
}

func TestRecordRoundWithError(t *testing.T) {
	mm := &ModelMetrics{
		ModelID:      "grok",
		RoundMetrics: make([]*RoundMetrics, 0),
		Errors:       make([]string, 0),
	}

	testErr := errors.New("test error")
	mm.RecordRound(1, 1*time.Second, 0, 0, testErr)

	if len(mm.Errors) != 1 {
		t.Fatalf("Expected 1 error, got %d", len(mm.Errors))
	}

	if mm.Errors[0] != testErr.Error() {
		t.Errorf("Expected error '%s', got '%s'", testErr.Error(), mm.Errors[0])
	}

	if mm.RoundMetrics[0].Error != testErr.Error() {
		t.Errorf("Expected round error '%s', got '%s'", testErr.Error(), mm.RoundMetrics[0].Error)
	}
}

func TestRecordRanking(t *testing.T) {
	mm := &ModelMetrics{
		ModelID: "grok",
	}

	mm.RecordRanking(500*time.Millisecond, 50, 25)

	if mm.RankingTime != 500*time.Millisecond {
		t.Errorf("Expected ranking time 500ms, got %v", mm.RankingTime)
	}

	if mm.RankingTokens.Input != 50 {
		t.Errorf("Expected 50 ranking input tokens, got %d", mm.RankingTokens.Input)
	}

	if mm.RankingTokens.Output != 25 {
		t.Errorf("Expected 25 ranking output tokens, got %d", mm.RankingTokens.Output)
	}

	if mm.TotalTokens.Input != 50 {
		t.Errorf("Expected total input 50, got %d", mm.TotalTokens.Input)
	}

	if mm.TotalTokens.Output != 25 {
		t.Errorf("Expected total output 25, got %d", mm.TotalTokens.Output)
	}
}

func TestComplete(t *testing.T) {
	rm := NewRequestMetrics("test-123", "Test", 1, 1)

	if !rm.EndTime.IsZero() {
		t.Error("EndTime should be zero initially")
	}

	rm.Complete("grok")

	if rm.EndTime.IsZero() {
		t.Error("EndTime should be set after Complete")
	}

	if rm.Winner != "grok" {
		t.Errorf("Expected winner 'grok', got %s", rm.Winner)
	}
}

func TestDuration(t *testing.T) {
	rm := NewRequestMetrics("test-123", "Test", 1, 1)

	// Before completion, should return time since start
	time.Sleep(10 * time.Millisecond)
	duration := rm.Duration()
	if duration < 10*time.Millisecond {
		t.Error("Duration should be at least 10ms")
	}

	// After completion
	rm.Complete("grok")
	completedDuration := rm.Duration()
	if completedDuration == 0 {
		t.Error("Duration should not be zero after completion")
	}
}

func TestSummary(t *testing.T) {
	rm := NewRequestMetrics("test-123", "What is AI?", 3, 4)

	mm1 := rm.AddModelMetrics("grok")
	mm1.RecordRound(1, 1*time.Second, 100, 50, nil)

	mm2 := rm.AddModelMetrics("gpt")
	mm2.RecordRound(1, 2*time.Second, 200, 100, nil)

	rm.Complete("grok")

	summary := rm.Summary()

	if summary["request_id"] != "test-123" {
		t.Errorf("Expected request_id 'test-123', got %v", summary["request_id"])
	}

	if summary["num_rounds"] != 3 {
		t.Errorf("Expected num_rounds 3, got %v", summary["num_rounds"])
	}

	if summary["num_models"] != 4 {
		t.Errorf("Expected num_models 4, got %v", summary["num_models"])
	}

	if summary["total_tokens_in"] != int64(300) {
		t.Errorf("Expected total_tokens_in 300, got %v", summary["total_tokens_in"])
	}

	if summary["total_tokens_out"] != int64(150) {
		t.Errorf("Expected total_tokens_out 150, got %v", summary["total_tokens_out"])
	}

	if summary["error_count"] != 0 {
		t.Errorf("Expected error_count 0, got %v", summary["error_count"])
	}

	if summary["winner"] != "grok" {
		t.Errorf("Expected winner 'grok', got %v", summary["winner"])
	}
}

func TestConcurrentAccess(t *testing.T) {
	rm := NewRequestMetrics("test-123", "Test", 1, 2)

	mm1 := rm.AddModelMetrics("grok")
	mm2 := rm.AddModelMetrics("gpt")

	// Simulate concurrent recording
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 10; i++ {
			mm1.RecordRound(i, 1*time.Second, 100, 50, nil)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			mm2.RecordRound(i, 1*time.Second, 100, 50, nil)
		}
		done <- true
	}()

	<-done
	<-done

	if len(mm1.RoundMetrics) != 10 {
		t.Errorf("Expected 10 rounds for mm1, got %d", len(mm1.RoundMetrics))
	}

	if len(mm2.RoundMetrics) != 10 {
		t.Errorf("Expected 10 rounds for mm2, got %d", len(mm2.RoundMetrics))
	}
}
