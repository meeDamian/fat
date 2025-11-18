# Archiver Package

Automatic background archival system for organizing answer folders by age.

## Overview

The archiver runs as a background goroutine that executes every hour to organize the `answers/` directory structure based on folder modification times.

## Directory Structure

```
answers/
├── X/                    # Fresh folders (< 1 week old)
├── recent/
│   └── Y/               # Recent folders (1 week - 1 month old)
└── archive/
    ├── 2025-01/
    │   └── Z/           # Archived folders (> 1 month old)
    └── 2025-02/
```

## Rules

1. **Folders older than 1 month** → `answers/archive/YYYY-MM/X`
   - Automatically organized into year-month subdirectories
   - Preserves all folder contents

2. **Folders older than 1 week** (but < 1 month) → `answers/recent/X`
   - Moves from `answers/` to `answers/recent/`
   - Keeps them accessible but out of the main directory

3. **Fresh folders** (< 1 week) → Stay in `answers/X`
   - Remain in the root answers directory for easy access

## Behavior

- **Runs immediately** on application startup
- **Runs every hour** thereafter via a background ticker
- **Checks both** `answers/` and `answers/recent/` directories
- **Skips duplicates** - if destination already exists, logs warning and skips
- **Preserves contents** - folder contents are fully preserved during moves
- **Creates directories** - automatically creates `recent/` and `archive/` as needed

## Age Calculation

Folders are aged based on their **modification time** (`os.FileInfo.ModTime()`), which updates when:
- Files are created/modified within the folder
- The folder itself is modified

## Usage

### Start the Archiver

In your `main.go`:

```go
import "github.com/meedamian/fat/internal/archiver"

func main() {
    logger := slog.Default()
    
    // Start background archiver (runs immediately, then every hour)
    archiver.StartBackgroundArchiver(logger)
    
    // Continue with application startup...
}
```

### Manual Archive Operation

```go
import "github.com/meedamian/fat/internal/archiver"

// Manually trigger archival (useful for testing or admin commands)
if err := archiver.ArchiveOldFolders(logger); err != nil {
    log.Printf("Archive failed: %v", err)
}
```

## Logging

The archiver logs all operations at appropriate levels:

- **INFO**: Successful moves with source/destination paths
- **WARN**: Duplicate destinations (skipped operations)
- **ERROR**: Failed directory operations
- **DEBUG**: Age thresholds and scan start times

Example log output:
```
level=INFO msg="starting background archiver" interval=1h0m0s
level=INFO msg="moved to recent" from=answers/1732806123_debate to=answers/recent/1732806123_debate
level=INFO msg="moved to archive" from=answers/recent/1730214123_old to=answers/archive/2024-10/1730214123_old mod_time=2024-10-29T10:35:23Z
```

## Testing

The package includes comprehensive tests:

```bash
go test ./internal/archiver/...
```

Tests cover:
- Moving folders to recent
- Moving folders to archive with YYYY-MM organization
- Duplicate handling
- Content preservation
- Proper directory creation

## Implementation Details

- Uses `os.Rename()` for atomic moves (same filesystem)
- Falls back gracefully if directories don't exist yet
- Thread-safe via Go's goroutine scheduler
- No external dependencies beyond standard library
- Testable via `*WithBase()` helper functions that accept custom base directories
