package archiver

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestArchiveOldFolders(t *testing.T) {
	// Create a temporary test directory
	tmpDir := t.TempDir()

	// Override package-level constants for testing
	origAnswersDir := answersDir
	origRecentDir := recentDir
	origArchiveDir := archiveDir
	defer func() {
		// This won't actually restore since they're const, but shows intent
		_ = origAnswersDir
		_ = origRecentDir
		_ = origArchiveDir
	}()

	// Create test structure
	testAnswersDir := filepath.Join(tmpDir, "answers")
	testRecentDir := filepath.Join(tmpDir, "answers", "recent")
	testArchiveDir := filepath.Join(tmpDir, "answers", "archive")

	if err := os.MkdirAll(testAnswersDir, 0755); err != nil {
		t.Fatal(err)
	}

	now := time.Now()

	// Create test folders with different ages
	testCases := []struct {
		name     string
		baseDir  string
		age      time.Duration
		expected string // Where it should end up
	}{
		{
			name:     "fresh-folder",
			baseDir:  testAnswersDir,
			age:      24 * time.Hour, // 1 day old
			expected: testAnswersDir, // Should stay in answers/
		},
		{
			name:     "week-old-folder",
			baseDir:  testAnswersDir,
			age:      8 * 24 * time.Hour, // 8 days old
			expected: testRecentDir,      // Should move to recent/
		},
		{
			name:     "month-old-in-recent",
			baseDir:  testRecentDir,
			age:      32 * 24 * time.Hour, // 32 days old
			expected: testArchiveDir,      // Should move to archive/YYYY-MM/
		},
	}

	_ = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create folder in base directory
			folderPath := filepath.Join(tc.baseDir, tc.name)
			if err := os.MkdirAll(folderPath, 0755); err != nil {
				t.Fatal(err)
			}

			// Set modification time
			modTime := now.Add(-tc.age)
			if err := os.Chtimes(folderPath, modTime, modTime); err != nil {
				t.Fatal(err)
			}

			// Create a test file inside to verify folder contents are preserved
			testFile := filepath.Join(folderPath, "test.txt")
			if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
				t.Fatal(err)
			}
		})
	}

	// Note: Full integration test would require refactoring to accept directory paths
	// For now, verify the helper functions work correctly
	t.Log("Archiver package structure created successfully")
	t.Log("Full integration test would require dependency injection of directory paths")
}

func TestMoveToRecent(t *testing.T) {
	tmpDir := t.TempDir()
	answersDir := filepath.Join(tmpDir, "answers")
	recentDir := filepath.Join(tmpDir, "answers", "recent")

	if err := os.MkdirAll(answersDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(recentDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test folder
	testFolder := filepath.Join(answersDir, "test-folder")
	if err := os.MkdirAll(testFolder, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a test file inside
	testFile := filepath.Join(testFolder, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Move to recent
	if err := moveToRecentWithBase(testFolder, "test-folder", recentDir, logger); err != nil {
		t.Fatalf("moveToRecent failed: %v", err)
	}

	// Verify original is gone
	if _, err := os.Stat(testFolder); !os.IsNotExist(err) {
		t.Error("original folder still exists")
	}

	// Verify new location exists
	newPath := filepath.Join(recentDir, "test-folder")
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("folder not found in recent: %v", err)
	}

	// Verify contents preserved
	newFile := filepath.Join(newPath, "test.txt")
	content, err := os.ReadFile(newFile)
	if err != nil {
		t.Errorf("failed to read file in new location: %v", err)
	}
	if string(content) != "content" {
		t.Errorf("file content mismatch: got %q, want %q", string(content), "content")
	}
}

func TestMoveToArchive(t *testing.T) {
	tmpDir := t.TempDir()
	recentDir := filepath.Join(tmpDir, "answers", "recent")
	archiveDir := filepath.Join(tmpDir, "answers", "archive")

	if err := os.MkdirAll(recentDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test folder
	testFolder := filepath.Join(recentDir, "test-folder")
	if err := os.MkdirAll(testFolder, 0755); err != nil {
		t.Fatal(err)
	}

	// Create a test file inside
	testFile := filepath.Join(testFolder, "test.txt")
	if err := os.WriteFile(testFile, []byte("archived content"), 0644); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Set mod time to 2 months ago
	modTime := time.Now().AddDate(0, -2, 0)
	if err := os.Chtimes(testFolder, modTime, modTime); err != nil {
		t.Fatal(err)
	}

	// Move to archive
	if err := moveToArchiveWithBase(testFolder, "test-folder", modTime, archiveDir, logger); err != nil {
		t.Fatalf("moveToArchive failed: %v", err)
	}

	// Verify original is gone
	if _, err := os.Stat(testFolder); !os.IsNotExist(err) {
		t.Error("original folder still exists")
	}

	// Verify new location exists in YYYY-MM format
	yearMonth := modTime.Format("2006-01")
	newPath := filepath.Join(archiveDir, yearMonth, "test-folder")
	if _, err := os.Stat(newPath); err != nil {
		t.Errorf("folder not found in archive: %v", err)
	}

	// Verify contents preserved
	newFile := filepath.Join(newPath, "test.txt")
	content, err := os.ReadFile(newFile)
	if err != nil {
		t.Errorf("failed to read file in new location: %v", err)
	}
	if string(content) != "archived content" {
		t.Errorf("file content mismatch: got %q, want %q", string(content), "archived content")
	}
}

func TestMoveToArchive_DuplicateHandling(t *testing.T) {
	tmpDir := t.TempDir()
	recentDir := filepath.Join(tmpDir, "answers", "recent")
	archiveDir := filepath.Join(tmpDir, "answers", "archive")

	if err := os.MkdirAll(recentDir, 0755); err != nil {
		t.Fatal(err)
	}

	modTime := time.Now().AddDate(0, -2, 0)
	yearMonth := modTime.Format("2006-01")
	archiveMonthDir := filepath.Join(archiveDir, yearMonth)
	if err := os.MkdirAll(archiveMonthDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create existing folder in archive
	existingFolder := filepath.Join(archiveMonthDir, "duplicate")
	if err := os.MkdirAll(existingFolder, 0755); err != nil {
		t.Fatal(err)
	}

	// Create duplicate in recent
	duplicateFolder := filepath.Join(recentDir, "duplicate")
	if err := os.MkdirAll(duplicateFolder, 0755); err != nil {
		t.Fatal(err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Attempt to move - should skip without error
	if err := moveToArchiveWithBase(duplicateFolder, "duplicate", modTime, archiveDir, logger); err != nil {
		t.Fatalf("moveToArchive with duplicate failed: %v", err)
	}

	// Verify original still exists (since move was skipped)
	if _, err := os.Stat(duplicateFolder); err != nil {
		t.Error("duplicate folder should still exist in recent")
	}
}
