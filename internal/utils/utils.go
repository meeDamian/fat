package utils

import (
	"fmt"
	"log/slog"
	"os"
	"time"
)

const answersDir = "answers"

var startTS int64

func SetStartTS(ts int64) {
	startTS = ts
}

// Log writes a conversation entry to a log file
//
// Deprecated: Use database storage instead. This function will be removed.
func Log(questionTS int64, logType, modelName, prompt, response string) error {
	// Create timestamp-specific directory
	tsDir := fmt.Sprintf("%s/%d", answersDir, questionTS)
	if err := os.MkdirAll(tsDir, 0755); err != nil {
		slog.Error("failed to create timestamp directory",
			slog.String("dir", tsDir),
			slog.Any("error", err))
		return fmt.Errorf("failed to create timestamp directory: %w", err)
	}

	diff := time.Now().Unix() - questionTS
	diffStr := fmt.Sprintf("%04d", diff)
	filename := fmt.Sprintf("%s/%s_%s_%s.log", tsDir, diffStr, logType, modelName)

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("failed to open log file",
			slog.String("filename", filename),
			slog.Any("error", err))
		return fmt.Errorf("failed to open log file %s: %w", filename, err)
	}
	defer file.Close()

	entry := fmt.Sprintf("=== PROMPT ===\n\n%s\n\n=== AGENT RESPONSE ===\n\n%s\n\n", prompt, response)
	if _, err := file.WriteString(entry); err != nil {
		slog.Error("failed to write to log file",
			slog.String("filename", filename),
			slog.Any("error", err))
		return fmt.Errorf("failed to write to log file %s: %w", filename, err)
	}

	slog.Debug("logged conversation",
		slog.String("filename", filename),
		slog.String("model", modelName),
		slog.String("type", logType))

	return nil
}

// LogCancellation creates an empty marker file to indicate a cancelled request
func LogCancellation(questionTS int64) error {
	// Create timestamp-specific directory
	tsDir := fmt.Sprintf("%s/%d", answersDir, questionTS)
	if err := os.MkdirAll(tsDir, 0755); err != nil {
		slog.Error("failed to create timestamp directory",
			slog.String("dir", tsDir),
			slog.Any("error", err))
		return fmt.Errorf("failed to create timestamp directory: %w", err)
	}

	diff := time.Now().Unix() - questionTS
	diffStr := fmt.Sprintf("%04d", diff)
	filename := fmt.Sprintf("%s/%s_CANCELLED", tsDir, diffStr)

	// Create empty file
	file, err := os.Create(filename)
	if err != nil {
		slog.Error("failed to create cancellation marker",
			slog.String("filename", filename),
			slog.Any("error", err))
		return fmt.Errorf("failed to create cancellation marker %s: %w", filename, err)
	}
	file.Close()

	slog.Info("created cancellation marker",
		slog.String("filename", filename))

	return nil
}
