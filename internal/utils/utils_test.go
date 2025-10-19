package utils

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLog(t *testing.T) {
	// Create temp directory for test
	tempDir := "test_answers"
	defer os.RemoveAll(tempDir)

	questionTS := time.Now().Unix()
	logType := "R1"
	modelName := "test-model"
	prompt := "Test prompt"
	response := "Test response"

	// Temporarily change to temp dir
	origWd, _ := os.Getwd()
	testDir, _ := os.MkdirTemp("", "fat_test")
	defer os.RemoveAll(testDir)
	os.Chdir(testDir)
	defer os.Chdir(origWd)

	err := Log(questionTS, logType, modelName, prompt, response)
	if err != nil {
		t.Fatalf("Log failed: %v", err)
	}

	// Verify directory was created
	if _, err := os.Stat(answersDir); os.IsNotExist(err) {
		t.Error("Answers directory was not created")
	}

	// Verify file was created
	files, err := filepath.Glob(filepath.Join(answersDir, "*.log"))
	if err != nil {
		t.Fatalf("Failed to glob files: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 log file, got %d", len(files))
	}

	// Read and verify content
	if len(files) > 0 {
		content, err := os.ReadFile(files[0])
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}

		contentStr := string(content)
		if !strings.Contains(contentStr, "=== PROMPT ===") {
			t.Error("Log file missing prompt header")
		}

		if !strings.Contains(contentStr, "=== AGENT RESPONSE ===") {
			t.Error("Log file missing response header")
		}

		if !strings.Contains(contentStr, prompt) {
			t.Error("Log file missing prompt content")
		}

		if !strings.Contains(contentStr, response) {
			t.Error("Log file missing response content")
		}
	}
}

func TestLogMultipleCalls(t *testing.T) {
	origWd, _ := os.Getwd()
	testDir, _ := os.MkdirTemp("", "fat_test_multi")
	defer os.RemoveAll(testDir)
	os.Chdir(testDir)
	defer os.Chdir(origWd)

	questionTS := time.Now().Unix()

	// Log multiple times
	for i := 0; i < 3; i++ {
		err := Log(questionTS, "R1", "model", "prompt", "response")
		if err != nil {
			t.Fatalf("Log call %d failed: %v", i, err)
		}
	}

	// Verify file exists and has multiple entries
	files, err := filepath.Glob(filepath.Join(answersDir, "*.log"))
	if err != nil {
		t.Fatalf("Failed to glob files: %v", err)
	}

	if len(files) != 1 {
		t.Errorf("Expected 1 log file, got %d", len(files))
	}

	if len(files) > 0 {
		content, err := os.ReadFile(files[0])
		if err != nil {
			t.Fatalf("Failed to read log file: %v", err)
		}

		// Should have 3 sets of headers
		contentStr := string(content)
		promptCount := strings.Count(contentStr, "=== PROMPT ===")
		if promptCount != 3 {
			t.Errorf("Expected 3 prompt headers, got %d", promptCount)
		}
	}
}

func TestLoadRates(t *testing.T) {
	ctx := context.Background()

	// Test with no rates.json file
	os.Remove("rates.json")

	rates := LoadRates(ctx)

	if rates == nil {
		t.Fatal("LoadRates returned nil")
	}

	// Should have default rates for all models
	expectedModels := []string{"grok-4-fast", "gpt-5-mini", "claude-3-5-haiku-20241022", "gemini-2.5-flash"}
	for _, model := range expectedModels {
		if _, ok := rates[model]; !ok {
			t.Errorf("Missing rate for model %s", model)
		}
	}

	// Verify rate structure
	for model, rate := range rates {
		if rate.TS == 0 {
			t.Errorf("Model %s has zero timestamp", model)
		}
		if rate.In == 0 {
			t.Errorf("Model %s has zero input rate", model)
		}
		if rate.Out == 0 {
			t.Errorf("Model %s has zero output rate", model)
		}
	}
}

func TestGetDefaultRates(t *testing.T) {
	rates := getDefaultRates()

	if len(rates) != 4 {
		t.Errorf("Expected 4 default rates, got %d", len(rates))
	}

	// Check specific rates
	if grokRate, ok := rates["grok-4-fast"]; ok {
		if grokRate.In != 0.20 {
			t.Errorf("Expected Grok input rate 0.20, got %f", grokRate.In)
		}
		if grokRate.Out != 0.50 {
			t.Errorf("Expected Grok output rate 0.50, got %f", grokRate.Out)
		}
	} else {
		t.Error("Missing Grok rate")
	}

	if gptRate, ok := rates["gpt-5-mini"]; ok {
		if gptRate.In != 0.25 {
			t.Errorf("Expected GPT input rate 0.25, got %f", gptRate.In)
		}
		if gptRate.Out != 2.00 {
			t.Errorf("Expected GPT output rate 2.00, got %f", gptRate.Out)
		}
	} else {
		t.Error("Missing GPT rate")
	}
}

func TestSetStartTS(t *testing.T) {
	testTS := int64(9876543210)
	SetStartTS(testTS)

	if startTS != testTS {
		t.Errorf("Expected startTS %d, got %d", testTS, startTS)
	}
}
