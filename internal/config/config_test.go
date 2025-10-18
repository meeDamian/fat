package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Clear any existing env vars
	os.Unsetenv("FAT_SERVER_ADDR")
	os.Unsetenv("FAT_MODEL_TIMEOUT")
	os.Unsetenv("FAT_LOG_LEVEL")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check defaults
	if cfg.ServerAddress != ":4444" {
		t.Errorf("Expected default ServerAddress ':4444', got %s", cfg.ServerAddress)
	}

	if cfg.ModelRequestTimeout != 30*time.Second {
		t.Errorf("Expected default ModelRequestTimeout 30s, got %v", cfg.ModelRequestTimeout)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("Expected default LogLevel 'info', got %s", cfg.LogLevel)
	}
}

func TestLoadWithEnvVars(t *testing.T) {
	os.Setenv("FAT_SERVER_ADDR", ":8080")
	os.Setenv("FAT_MODEL_TIMEOUT", "60s")
	os.Setenv("FAT_LOG_LEVEL", "debug")
	defer func() {
		os.Unsetenv("FAT_SERVER_ADDR")
		os.Unsetenv("FAT_MODEL_TIMEOUT")
		os.Unsetenv("FAT_LOG_LEVEL")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.ServerAddress != ":8080" {
		t.Errorf("Expected ServerAddress ':8080', got %s", cfg.ServerAddress)
	}

	if cfg.ModelRequestTimeout != 60*time.Second {
		t.Errorf("Expected ModelRequestTimeout 60s, got %v", cfg.ModelRequestTimeout)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("Expected LogLevel 'debug', got %s", cfg.LogLevel)
	}
}

func TestLoadWithInvalidTimeout(t *testing.T) {
	os.Setenv("FAT_MODEL_TIMEOUT", "invalid")
	defer os.Unsetenv("FAT_MODEL_TIMEOUT")

	_, err := Load()
	if err == nil {
		t.Error("Expected error for invalid timeout, got nil")
	}
}

func TestEnvOrDefault(t *testing.T) {
	os.Unsetenv("TEST_VAR")

	result := envOrDefault("TEST_VAR", "default")
	if result != "default" {
		t.Errorf("Expected 'default', got %s", result)
	}

	os.Setenv("TEST_VAR", "custom")
	defer os.Unsetenv("TEST_VAR")

	result = envOrDefault("TEST_VAR", "default")
	if result != "custom" {
		t.Errorf("Expected 'custom', got %s", result)
	}
}

func TestNewLogger(t *testing.T) {
	tests := []struct {
		level     string
		shouldErr bool
	}{
		{"debug", false},
		{"info", false},
		{"warn", false},
		{"error", false},
		{"", false}, // Empty defaults to info
		{"invalid", true},
	}

	for _, tt := range tests {
		logger, err := NewLogger(tt.level)

		if tt.shouldErr {
			if err == nil {
				t.Errorf("Expected error for level '%s', got nil", tt.level)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for level '%s': %v", tt.level, err)
			}
			if logger == nil {
				t.Errorf("Expected logger for level '%s', got nil", tt.level)
			}
		}
	}
}
