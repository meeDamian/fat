package db

import (
	"context"
	"os"
	"testing"
	"time"

	"log/slog"
)

func TestNew(t *testing.T) {
	dbPath := "test.db"
	defer os.Remove(dbPath)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db, err := New(dbPath, logger)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	if db.conn == nil {
		t.Error("Database connection is nil")
	}
}

func TestSaveRequest(t *testing.T) {
	dbPath := "test_request.db"
	defer os.Remove(dbPath)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db, err := New(dbPath, logger)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	req := Request{
		ID:              "test-123",
		Question:        "What is AI?",
		NumRounds:       3,
		NumModels:       4,
		WinnerModel:     "grok",
		TotalDurationMs: 5000,
		TotalTokensIn:   1000,
		TotalTokensOut:  500,
		TotalCost:       0.05,
		ErrorCount:      0,
	}

	if err := db.SaveRequest(ctx, req); err != nil {
		t.Fatalf("Failed to save request: %v", err)
	}

	// Verify it was saved
	requests, err := db.GetRecentRequests(ctx, 1)
	if err != nil {
		t.Fatalf("Failed to get recent requests: %v", err)
	}

	if len(requests) != 1 {
		t.Fatalf("Expected 1 request, got %d", len(requests))
	}

	if requests[0].ID != req.ID {
		t.Errorf("Expected ID %s, got %s", req.ID, requests[0].ID)
	}

	if requests[0].Question != req.Question {
		t.Errorf("Expected question %s, got %s", req.Question, requests[0].Question)
	}
}

func TestSaveModelRound(t *testing.T) {
	dbPath := "test_round.db"
	defer os.Remove(dbPath)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db, err := New(dbPath, logger)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// First save a request
	req := Request{
		ID:              "test-456",
		Question:        "Test question",
		NumRounds:       1,
		NumModels:       1,
		WinnerModel:     "grok",
		TotalDurationMs: 1000,
		TotalTokensIn:   100,
		TotalTokensOut:  50,
		TotalCost:       0.01,
		ErrorCount:      0,
	}
	if err := db.SaveRequest(ctx, req); err != nil {
		t.Fatalf("Failed to save request: %v", err)
	}

	// Now save a model round
	mr := ModelRound{
		RequestID:  "test-456",
		ModelID:    "grok",
		ModelName:  "grok-4-fast",
		Round:      1,
		DurationMs: 1000,
		TokensIn:   100,
		TokensOut:  50,
		Cost:       0.01,
		Error:      "",
	}

	if err := db.SaveModelRound(ctx, mr); err != nil {
		t.Fatalf("Failed to save model round: %v", err)
	}
}

func TestUpdateModelStats(t *testing.T) {
	dbPath := "test_stats.db"
	defer os.Remove(dbPath)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db, err := New(dbPath, logger)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Update stats for a model
	err = db.UpdateModelStats(ctx, "grok", "grok-4-fast", true, 100, 50, 0.01, 1000)
	if err != nil {
		t.Fatalf("Failed to update model stats: %v", err)
	}

	// Get the stats back
	stats, err := db.GetModelStats(ctx, "grok")
	if err != nil {
		t.Fatalf("Failed to get model stats: %v", err)
	}

	if stats == nil {
		t.Fatal("Expected stats, got nil")
	}

	if stats.ModelID != "grok" {
		t.Errorf("Expected model ID 'grok', got %s", stats.ModelID)
	}

	if stats.TotalRequests != 1 {
		t.Errorf("Expected 1 request, got %d", stats.TotalRequests)
	}

	if stats.TotalWins != 1 {
		t.Errorf("Expected 1 win, got %d", stats.TotalWins)
	}

	if stats.TotalTokensIn != 100 {
		t.Errorf("Expected 100 tokens in, got %d", stats.TotalTokensIn)
	}

	// Update again (should increment)
	err = db.UpdateModelStats(ctx, "grok", "grok-4-fast", false, 200, 100, 0.02, 2000)
	if err != nil {
		t.Fatalf("Failed to update model stats second time: %v", err)
	}

	stats, err = db.GetModelStats(ctx, "grok")
	if err != nil {
		t.Fatalf("Failed to get model stats: %v", err)
	}

	if stats.TotalRequests != 2 {
		t.Errorf("Expected 2 requests, got %d", stats.TotalRequests)
	}

	if stats.TotalWins != 1 {
		t.Errorf("Expected 1 win (no increment), got %d", stats.TotalWins)
	}

	if stats.TotalTokensIn != 300 {
		t.Errorf("Expected 300 tokens in, got %d", stats.TotalTokensIn)
	}
}

func TestGetAllModelStats(t *testing.T) {
	dbPath := "test_all_stats.db"
	defer os.Remove(dbPath)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db, err := New(dbPath, logger)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Add stats for multiple models
	models := []struct {
		id   string
		name string
	}{
		{"grok", "grok-4-fast"},
		{"gpt", "gpt-5-mini"},
		{"claude", "claude-3.5-haiku"},
	}

	for _, m := range models {
		err = db.UpdateModelStats(ctx, m.id, m.name, false, 100, 50, 0.01, 1000)
		if err != nil {
			t.Fatalf("Failed to update stats for %s: %v", m.id, err)
		}
	}

	// Get all stats
	allStats, err := db.GetAllModelStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get all model stats: %v", err)
	}

	if len(allStats) != 3 {
		t.Errorf("Expected 3 model stats, got %d", len(allStats))
	}
}

func TestGetRecentRequests(t *testing.T) {
	dbPath := "test_recent.db"
	defer os.Remove(dbPath)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db, err := New(dbPath, logger)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// Add multiple requests
	for i := 0; i < 5; i++ {
		req := Request{
			ID:              string(rune('a' + i)),
			Question:        "Question " + string(rune('0'+i)),
			NumRounds:       1,
			NumModels:       1,
			WinnerModel:     "grok",
			TotalDurationMs: 1000,
			TotalTokensIn:   100,
			TotalTokensOut:  50,
			TotalCost:       0.01,
			ErrorCount:      0,
		}
		if err := db.SaveRequest(ctx, req); err != nil {
			t.Fatalf("Failed to save request %d: %v", i, err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Get recent 3
	recent, err := db.GetRecentRequests(ctx, 3)
	if err != nil {
		t.Fatalf("Failed to get recent requests: %v", err)
	}

	if len(recent) != 3 {
		t.Errorf("Expected 3 recent requests, got %d", len(recent))
	}

	// Should be in reverse chronological order
	if len(recent) >= 2 && recent[0].CreatedAt.Before(recent[1].CreatedAt) {
		t.Error("Requests not in reverse chronological order")
	}
}

func TestSaveRanking(t *testing.T) {
	dbPath := "test_ranking.db"
	defer os.Remove(dbPath)

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	db, err := New(dbPath, logger)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	ctx := context.Background()

	// First save a request
	req := Request{
		ID:              "test-789",
		Question:        "Test ranking",
		NumRounds:       1,
		NumModels:       3,
		WinnerModel:     "grok",
		TotalDurationMs: 1000,
		TotalTokensIn:   100,
		TotalTokensOut:  50,
		TotalCost:       0.01,
		ErrorCount:      0,
	}
	if err := db.SaveRequest(ctx, req); err != nil {
		t.Fatalf("Failed to save request: %v", err)
	}

	// Save ranking
	ranking := Ranking{
		RequestID:    "test-789",
		RankerModel:  "grok",
		RankedModels: `["grok","gpt","claude"]`,
		DurationMs:   500,
		TokensIn:     50,
		TokensOut:    25,
		Cost:         0.005,
	}

	if err := db.SaveRanking(ctx, ranking); err != nil {
		t.Fatalf("Failed to save ranking: %v", err)
	}
}
