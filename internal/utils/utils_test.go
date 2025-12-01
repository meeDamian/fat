package utils

import (
	"fmt"
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

	// Verify file was created in timestamp subdirectory
	tsDir := filepath.Join(answersDir, fmt.Sprintf("%d", questionTS))
	files, err := filepath.Glob(filepath.Join(tsDir, "*.log"))
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

	// Verify file exists and has multiple entries in timestamp subdirectory
	tsDir := filepath.Join(answersDir, fmt.Sprintf("%d", questionTS))
	files, err := filepath.Glob(filepath.Join(tsDir, "*.log"))
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

func TestSetStartTS(t *testing.T) {
	testTS := int64(9876543210)
	SetStartTS(testTS)

	if startTS != testTS {
		t.Errorf("Expected startTS %d, got %d", testTS, startTS)
	}
}
