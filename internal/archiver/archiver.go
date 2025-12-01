// Package archiver handles moving old log files to archive directories.
//
// Deprecated: This package is deprecated in favor of database storage.
// It will be removed in a future release.
package archiver

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	answersDir = "answers"
	recentDir  = "answers/recent"
	archiveDir = "answers/archive"
)

// StartBackgroundArchiver starts a goroutine that runs archive operations every hour
func StartBackgroundArchiver(logger *slog.Logger) {
	logger.Info("starting background archiver", slog.Duration("interval", time.Hour))

	// Run immediately on startup
	if err := ArchiveOldFolders(logger); err != nil {
		logger.Error("initial archive run failed", slog.Any("error", err))
	}

	// Then run every hour
	ticker := time.NewTicker(time.Hour)
	go func() {
		for range ticker.C {
			if err := ArchiveOldFolders(logger); err != nil {
				logger.Error("archive run failed", slog.Any("error", err))
			}
		}
	}()
}

// ArchiveOldFolders moves folders based on their age:
// - Folders older than 1 month → answers/archive/YYYY-MM/
// - Folders older than 1 week → answers/recent/
func ArchiveOldFolders(logger *slog.Logger) error {
	now := time.Now()
	oneWeekAgo := now.AddDate(0, 0, -7)
	oneMonthAgo := now.AddDate(0, -1, 0)

	logger.Debug("starting archive scan",
		slog.Time("now", now),
		slog.Time("one_week_ago", oneWeekAgo),
		slog.Time("one_month_ago", oneMonthAgo))

	// Ensure archive and recent directories exist
	if err := os.MkdirAll(recentDir, 0755); err != nil {
		return fmt.Errorf("failed to create recent dir: %w", err)
	}
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return fmt.Errorf("failed to create archive dir: %w", err)
	}

	// Check folders in answers/recent/
	if err := processDirectory(recentDir, oneMonthAgo, logger, true); err != nil {
		logger.Error("failed to process recent directory", slog.Any("error", err))
	}

	// Check folders in answers/
	if err := processDirectory(answersDir, oneWeekAgo, logger, false); err != nil {
		logger.Error("failed to process answers directory", slog.Any("error", err))
	}

	return nil
}

// processDirectory scans a directory and moves old folders
func processDirectory(dirPath string, ageThreshold time.Time, logger *slog.Logger, isRecentDir bool) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Directory doesn't exist yet, that's fine
		}
		return fmt.Errorf("failed to read directory %s: %w", dirPath, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// Skip special directories
		name := entry.Name()
		if name == "recent" || name == "archive" || strings.HasPrefix(name, ".") {
			continue
		}

		fullPath := filepath.Join(dirPath, name)
		info, err := entry.Info()
		if err != nil {
			logger.Warn("failed to get file info",
				slog.String("path", fullPath),
				slog.Any("error", err))
			continue
		}

		modTime := info.ModTime()

		if isRecentDir {
			// From recent/ - move to archive if older than 1 month
			if modTime.Before(ageThreshold) {
				if err := moveToArchive(fullPath, name, modTime, logger); err != nil {
					logger.Error("failed to move to archive",
						slog.String("path", fullPath),
						slog.Any("error", err))
				}
			}
		} else {
			// From answers/ - move to recent if older than 1 week
			if modTime.Before(ageThreshold) {
				if err := moveToRecent(fullPath, name, logger); err != nil {
					logger.Error("failed to move to recent",
						slog.String("path", fullPath),
						slog.Any("error", err))
				}
			}
		}
	}

	return nil
}

// moveToArchive moves a folder to answers/archive/YYYY-MM/
func moveToArchive(srcPath, name string, modTime time.Time, logger *slog.Logger) error {
	return moveToArchiveWithBase(srcPath, name, modTime, archiveDir, logger)
}

// moveToArchiveWithBase is the testable version that accepts a base directory
func moveToArchiveWithBase(srcPath, name string, modTime time.Time, baseArchiveDir string, logger *slog.Logger) error {
	// Create YYYY-MM directory
	yearMonth := modTime.Format("2006-01")
	archiveMonthDir := filepath.Join(baseArchiveDir, yearMonth)
	if err := os.MkdirAll(archiveMonthDir, 0755); err != nil {
		return fmt.Errorf("failed to create archive month dir: %w", err)
	}

	destPath := filepath.Join(archiveMonthDir, name)

	// Check if destination already exists
	if _, err := os.Stat(destPath); err == nil {
		logger.Warn("destination already exists, skipping",
			slog.String("src", srcPath),
			slog.String("dest", destPath))
		return nil
	}

	// Move the folder
	if err := os.Rename(srcPath, destPath); err != nil {
		return fmt.Errorf("failed to rename: %w", err)
	}

	logger.Info("moved to archive",
		slog.String("from", srcPath),
		slog.String("to", destPath),
		slog.Time("mod_time", modTime))

	return nil
}

// moveToRecent moves a folder to answers/recent/
func moveToRecent(srcPath, name string, logger *slog.Logger) error {
	return moveToRecentWithBase(srcPath, name, recentDir, logger)
}

// moveToRecentWithBase is the testable version that accepts a base directory
func moveToRecentWithBase(srcPath, name string, baseRecentDir string, logger *slog.Logger) error {
	destPath := filepath.Join(baseRecentDir, name)

	// Check if destination already exists
	if _, err := os.Stat(destPath); err == nil {
		logger.Warn("destination already exists, skipping",
			slog.String("src", srcPath),
			slog.String("dest", destPath))
		return nil
	}

	// Move the folder
	if err := os.Rename(srcPath, destPath); err != nil {
		return fmt.Errorf("failed to rename: %w", err)
	}

	logger.Info("moved to recent",
		slog.String("from", srcPath),
		slog.String("to", destPath))

	return nil
}
