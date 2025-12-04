package db

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite database
type DB struct {
	conn   *sql.DB
	logger *slog.Logger
}

// New creates a new database connection and initializes schema
func New(dbPath string, logger *slog.Logger) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := conn.Exec("PRAGMA journal_mode=WAL"); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	db := &DB{
		conn:   conn,
		logger: logger,
	}

	if err := db.initSchema(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Run any pending migrations
	if err := db.RunMigrations(context.Background()); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// initSchema creates all necessary tables
func (db *DB) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS requests (
		id TEXT PRIMARY KEY,
		question TEXT NOT NULL,
		num_rounds INTEGER NOT NULL,
		num_models INTEGER NOT NULL,
		winner_model TEXT,
		total_duration_ms INTEGER,
		total_tokens_in INTEGER,
		total_tokens_out INTEGER,
		total_cost REAL,
		error_count INTEGER,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS model_rounds (
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

	CREATE TABLE IF NOT EXISTS rankings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		request_id TEXT NOT NULL,
		ranker_model TEXT NOT NULL,
		ranked_models TEXT NOT NULL, -- JSON array of model names in order
		duration_ms INTEGER NOT NULL,
		tokens_in INTEGER NOT NULL,
		tokens_out INTEGER NOT NULL,
		cost REAL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (request_id) REFERENCES requests(id)
	);

	CREATE TABLE IF NOT EXISTS model_stats (
		model_id TEXT PRIMARY KEY,
		model_name TEXT NOT NULL,
		total_requests INTEGER DEFAULT 0,
		total_wins INTEGER DEFAULT 0,
		total_tokens_in INTEGER DEFAULT 0,
		total_tokens_out INTEGER DEFAULT 0,
		total_cost REAL DEFAULT 0,
		avg_response_time_ms INTEGER DEFAULT 0,
		error_count INTEGER DEFAULT 0,
		last_used TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_requests_created ON requests(created_at);
	CREATE INDEX IF NOT EXISTS idx_model_rounds_request ON model_rounds(request_id);
	CREATE INDEX IF NOT EXISTS idx_model_rounds_model ON model_rounds(model_id);
	CREATE INDEX IF NOT EXISTS idx_model_rounds_model_round ON model_rounds(model_id, round);
	CREATE INDEX IF NOT EXISTS idx_rankings_request ON rankings(request_id);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// Request represents a complete request record
type Request struct {
	ID              string
	Question        string
	NumRounds       int
	NumModels       int
	WinnerModel     string
	TotalDurationMs int64
	TotalTokensIn   int64
	TotalTokensOut  int64
	TotalCost       float64
	ErrorCount      int
	CreatedAt       time.Time
}

// ModelRound represents a single model's performance in one round
type ModelRound struct {
	ID         int64
	RequestID  string
	ModelID    string
	ModelName  string
	Round      int
	DurationMs int64
	TokensIn   int64
	TokensOut  int64
	Cost       float64
	Error      string
	// Content fields (previously in RoundReply)
	Answer       string
	Rationale    string
	Discussion   string // JSON map of target_agent -> messages
	PrivateNotes string // Private notes (never shared with other agents)
	CreatedAt    time.Time
}

// Ranking represents a model's ranking of all agents
type Ranking struct {
	ID           int64
	RequestID    string
	RankerModel  string
	RankedModels string // JSON array
	DurationMs   int64
	TokensIn     int64
	TokensOut    int64
	Cost         float64
	CreatedAt    time.Time
}

// ModelStats represents aggregate statistics for a model
type ModelStats struct {
	ModelID           string
	ModelName         string
	TotalRequests     int64
	TotalWins         int64
	TotalTokensIn     int64
	TotalTokensOut    int64
	TotalCost         float64
	AvgResponseTimeMs int64
	ErrorCount        int64
	LastUsed          time.Time
	UpdatedAt         time.Time
}

// SaveRequest saves a complete request record
func (db *DB) SaveRequest(ctx context.Context, req Request) error {
	query := `
		INSERT INTO requests (
			id, question, num_rounds, num_models, winner_model,
			total_duration_ms, total_tokens_in, total_tokens_out,
			total_cost, error_count
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.conn.ExecContext(ctx, query,
		req.ID, req.Question, req.NumRounds, req.NumModels, req.WinnerModel,
		req.TotalDurationMs, req.TotalTokensIn, req.TotalTokensOut,
		req.TotalCost, req.ErrorCount,
	)

	if err != nil {
		return fmt.Errorf("failed to save request: %w", err)
	}

	db.logger.Debug("saved request to database",
		slog.String("request_id", req.ID),
		slog.Int64("duration_ms", req.TotalDurationMs))

	return nil
}

// SaveModelRound saves a model's performance and content in a single round
func (db *DB) SaveModelRound(ctx context.Context, mr ModelRound) error {
	query := `
		INSERT INTO model_rounds (
			request_id, model_id, model_name, round,
			duration_ms, tokens_in, tokens_out, cost, error,
			answer, rationale, discussion, private_notes
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(request_id, model_id, round) DO UPDATE SET
			duration_ms = excluded.duration_ms,
			tokens_in = excluded.tokens_in,
			tokens_out = excluded.tokens_out,
			cost = excluded.cost,
			error = excluded.error,
			answer = excluded.answer,
			rationale = excluded.rationale,
			discussion = excluded.discussion,
			private_notes = excluded.private_notes
	`

	_, err := db.conn.ExecContext(ctx, query,
		mr.RequestID, mr.ModelID, mr.ModelName, mr.Round,
		mr.DurationMs, mr.TokensIn, mr.TokensOut, mr.Cost, mr.Error,
		mr.Answer, mr.Rationale, mr.Discussion, mr.PrivateNotes,
	)

	if err != nil {
		return fmt.Errorf("failed to save model round: %w", err)
	}

	return nil
}

// SaveRanking saves a ranking record
func (db *DB) SaveRanking(ctx context.Context, r Ranking) error {
	query := `
		INSERT INTO rankings (
			request_id, ranker_model, ranked_models,
			duration_ms, tokens_in, tokens_out, cost
		) VALUES (?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.conn.ExecContext(ctx, query,
		r.RequestID, r.RankerModel, r.RankedModels,
		r.DurationMs, r.TokensIn, r.TokensOut, r.Cost,
	)

	if err != nil {
		return fmt.Errorf("failed to save ranking: %w", err)
	}

	return nil
}

// GetRoundReplies retrieves all round data for a request
func (db *DB) GetRoundReplies(ctx context.Context, requestID string) (map[string]map[int]ModelRound, error) {
	query := `
		SELECT id, request_id, model_id, model_name, round,
		       duration_ms, tokens_in, tokens_out, cost, error,
		       answer, rationale, discussion, COALESCE(private_notes, ''), created_at
		FROM model_rounds
		WHERE request_id = ?
		ORDER BY model_id, round
	`

	rows, err := db.conn.QueryContext(ctx, query, requestID)
	if err != nil {
		return nil, fmt.Errorf("failed to query round data: %w", err)
	}
	defer rows.Close()

	// Map structure: modelID -> round -> ModelRound
	replies := make(map[string]map[int]ModelRound)

	for rows.Next() {
		var mr ModelRound
		err := rows.Scan(
			&mr.ID, &mr.RequestID, &mr.ModelID, &mr.ModelName, &mr.Round,
			&mr.DurationMs, &mr.TokensIn, &mr.TokensOut, &mr.Cost, &mr.Error,
			&mr.Answer, &mr.Rationale, &mr.Discussion, &mr.PrivateNotes, &mr.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan round data: %w", err)
		}

		if replies[mr.ModelID] == nil {
			replies[mr.ModelID] = make(map[int]ModelRound)
		}
		replies[mr.ModelID][mr.Round] = mr
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating round data: %w", err)
	}

	return replies, nil
}

// UpdateModelStats updates aggregate statistics for a model
func (db *DB) UpdateModelStats(ctx context.Context, modelID, modelName string, won bool, tokensIn, tokensOut int64, cost float64, responseTimeMs int64) error {
	// Upsert model stats
	query := `
		INSERT INTO model_stats (
			model_id, model_name, total_requests, total_wins,
			total_tokens_in, total_tokens_out, total_cost,
			avg_response_time_ms, last_used, updated_at
		) VALUES (?, ?, 1, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(model_id) DO UPDATE SET
			total_requests = total_requests + 1,
			total_wins = total_wins + ?,
			total_tokens_in = total_tokens_in + ?,
			total_tokens_out = total_tokens_out + ?,
			total_cost = total_cost + ?,
			avg_response_time_ms = (avg_response_time_ms * total_requests + ?) / (total_requests + 1),
			last_used = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
	`

	winInt := 0
	if won {
		winInt = 1
	}

	_, err := db.conn.ExecContext(ctx, query,
		modelID, modelName, winInt, tokensIn, tokensOut, cost, responseTimeMs,
		winInt, tokensIn, tokensOut, cost, responseTimeMs,
	)

	if err != nil {
		return fmt.Errorf("failed to update model stats: %w", err)
	}

	return nil
}

// GetModelStats retrieves statistics for a specific model
func (db *DB) GetModelStats(ctx context.Context, modelID string) (*ModelStats, error) {
	query := `
		SELECT model_id, model_name, total_requests, total_wins,
			   total_tokens_in, total_tokens_out, total_cost,
			   avg_response_time_ms, error_count, last_used, updated_at
		FROM model_stats
		WHERE model_id = ?
	`

	var stats ModelStats
	err := db.conn.QueryRowContext(ctx, query, modelID).Scan(
		&stats.ModelID, &stats.ModelName, &stats.TotalRequests, &stats.TotalWins,
		&stats.TotalTokensIn, &stats.TotalTokensOut, &stats.TotalCost,
		&stats.AvgResponseTimeMs, &stats.ErrorCount, &stats.LastUsed, &stats.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get model stats: %w", err)
	}

	return &stats, nil
}

// GetAllModelStats retrieves statistics for all models
func (db *DB) GetAllModelStats(ctx context.Context) ([]ModelStats, error) {
	query := `
		SELECT model_id, model_name, total_requests, total_wins,
			   total_tokens_in, total_tokens_out, total_cost,
			   avg_response_time_ms, error_count, last_used, updated_at
		FROM model_stats
		ORDER BY total_requests DESC
	`

	rows, err := db.conn.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query model stats: %w", err)
	}
	defer rows.Close()

	var stats []ModelStats
	for rows.Next() {
		var s ModelStats
		if err := rows.Scan(
			&s.ModelID, &s.ModelName, &s.TotalRequests, &s.TotalWins,
			&s.TotalTokensIn, &s.TotalTokensOut, &s.TotalCost,
			&s.AvgResponseTimeMs, &s.ErrorCount, &s.LastUsed, &s.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan model stats: %w", err)
		}
		stats = append(stats, s)
	}

	return stats, rows.Err()
}

// GetRecentRequests retrieves the most recent N requests
func (db *DB) GetRecentRequests(ctx context.Context, limit int) ([]Request, error) {
	query := `
		SELECT id, question, num_rounds, num_models, winner_model,
			   total_duration_ms, total_tokens_in, total_tokens_out,
			   total_cost, error_count, created_at
		FROM requests
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := db.conn.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent requests: %w", err)
	}
	defer rows.Close()

	var requests []Request
	for rows.Next() {
		var r Request
		if err := rows.Scan(
			&r.ID, &r.Question, &r.NumRounds, &r.NumModels, &r.WinnerModel,
			&r.TotalDurationMs, &r.TotalTokensIn, &r.TotalTokensOut,
			&r.TotalCost, &r.ErrorCount, &r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan request: %w", err)
		}
		requests = append(requests, r)
	}

	return requests, rows.Err()
}
