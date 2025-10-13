package utils

import (
	"context"
	"testing"

	"github.com/meedamian/fat/internal/types"
)

func TestEstTokens(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"hello", 101},
		{"", 100},
		{"this is a longer text", 105},
	}
	for _, tt := range tests {
		result := EstTokens(tt.input)
		if result != tt.expected {
			t.Errorf("EstTokens(%q) = %d; want %d", tt.input, result, tt.expected)
		}
	}
}

func TestBuildContext(t *testing.T) {
	history := make(map[string][]types.Response)
	// Add test data
	context := BuildContext("What is Go?", history)
	if len(context) == 0 {
		t.Error("BuildContext returned empty string")
	}
}

func TestFetchRates(t *testing.T) {
	ctx := context.Background()
	rates, err := FetchRates(ctx)
	if err != nil {
		t.Errorf("FetchRates failed: %v", err)
	}
	if len(rates) == 0 {
		t.Error("FetchRates returned empty map")
	}
}
