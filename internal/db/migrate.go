package db

import (
	"context"
	"fmt"
)

// MigrateConsolidateRounds consolidates model_rounds and round_replies tables
func (db *DB) MigrateConsolidateRounds(ctx context.Context) error {
	db.logger.Info("starting database migration: consolidate rounds")

	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Step 1: Create new consolidated table
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS model_rounds_new (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		request_id TEXT NOT NULL,
		model_id TEXT NOT NULL,
		model_name TEXT NOT NULL,
		round INTEGER NOT NULL,
		duration_ms INTEGER NOT NULL,
		tokens_in INTEGER NOT NULL,
		tokens_out INTEGER NOT NULL,
		cost REAL,
		error TEXT,
		answer TEXT,
		rationale TEXT,
		discussion TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (request_id) REFERENCES requests(id),
		UNIQUE(request_id, model_id, round)
	);
	`
	if _, err := tx.ExecContext(ctx, createTableSQL); err != nil {
		return fmt.Errorf("failed to create new table: %w", err)
	}

	// Step 2: Migrate data from both tables
	migrateDataSQL := `
	INSERT INTO model_rounds_new (
		request_id, model_id, model_name, round,
		duration_ms, tokens_in, tokens_out, cost, error,
		answer, rationale, discussion, created_at
	)
	SELECT 
		mr.request_id, mr.model_id, mr.model_name, mr.round,
		mr.duration_ms, mr.tokens_in, mr.tokens_out, mr.cost, mr.error,
		COALESCE(rr.answer, ''), COALESCE(rr.rationale, ''), COALESCE(rr.discussion, ''),
		mr.created_at
	FROM model_rounds mr
	LEFT JOIN round_replies rr 
		ON mr.request_id = rr.request_id 
		AND mr.model_id = rr.model_id 
		AND mr.round = rr.round;
	`
	if _, err := tx.ExecContext(ctx, migrateDataSQL); err != nil {
		return fmt.Errorf("failed to migrate data: %w", err)
	}

	// Step 3: Drop old tables
	if _, err := tx.ExecContext(ctx, "DROP TABLE IF EXISTS round_replies"); err != nil {
		return fmt.Errorf("failed to drop round_replies: %w", err)
	}
	if _, err := tx.ExecContext(ctx, "DROP TABLE IF EXISTS model_rounds"); err != nil {
		return fmt.Errorf("failed to drop model_rounds: %w", err)
	}

	// Step 4: Rename new table
	if _, err := tx.ExecContext(ctx, "ALTER TABLE model_rounds_new RENAME TO model_rounds"); err != nil {
		return fmt.Errorf("failed to rename table: %w", err)
	}

	// Step 5: Recreate indexes
	indexesSQL := `
	CREATE INDEX IF NOT EXISTS idx_model_rounds_request ON model_rounds(request_id);
	CREATE INDEX IF NOT EXISTS idx_model_rounds_model ON model_rounds(model_id);
	CREATE INDEX IF NOT EXISTS idx_model_rounds_model_round ON model_rounds(model_id, round);
	`
	if _, err := tx.ExecContext(ctx, indexesSQL); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	db.logger.Info("database migration completed successfully")
	return nil
}

// getSchemaVersion retrieves the current schema version
func (db *DB) getSchemaVersion(ctx context.Context) (int, error) {
	// Create schema_version table if it doesn't exist
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY,
		applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.conn.ExecContext(ctx, createTableSQL); err != nil {
		return 0, fmt.Errorf("failed to create schema_version table: %w", err)
	}

	var version int
	err := db.conn.QueryRowContext(ctx, "SELECT COALESCE(MAX(version), 0) FROM schema_version").Scan(&version)
	if err != nil {
		return 0, fmt.Errorf("failed to get schema version: %w", err)
	}

	return version, nil
}

// setSchemaVersion sets the schema version
func (db *DB) setSchemaVersion(ctx context.Context, version int) error {
	_, err := db.conn.ExecContext(ctx, "INSERT INTO schema_version (version) VALUES (?)", version)
	if err != nil {
		return fmt.Errorf("failed to set schema version: %w", err)
	}
	return nil
}

// RunMigrations runs all pending migrations
func (db *DB) RunMigrations(ctx context.Context) error {
	version, err := db.getSchemaVersion(ctx)
	if err != nil {
		return err
	}

	db.logger.Info("current schema version", "version", version)

	if version < 1 {
		db.logger.Info("running migration: consolidate rounds")
		if err := db.MigrateConsolidateRounds(ctx); err != nil {
			return err
		}
		if err := db.setSchemaVersion(ctx, 1); err != nil {
			return err
		}
		db.logger.Info("migration completed", "new_version", 1)
	}

	return nil
}
