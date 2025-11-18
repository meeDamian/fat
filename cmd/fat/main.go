package main

import (
	"fmt"
	"log/slog"

	"github.com/meedamian/fat/internal/apikeys"
	"github.com/meedamian/fat/internal/archiver"
	"github.com/meedamian/fat/internal/config"
	"github.com/meedamian/fat/internal/db"
	"github.com/meedamian/fat/internal/models"
	"github.com/meedamian/fat/internal/server"
	"github.com/meedamian/fat/internal/types"
	"github.com/meedamian/fat/web"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}

	// Initialize logger
	logger, err := config.NewLogger(cfg.LogLevel)
	if err != nil {
		panic(fmt.Errorf("failed to create logger: %w", err))
	}

	// Load API keys
	logger.Info("loading API keys")
	allModels := make([]*types.ModelInfo, 0, len(models.AllModels))
	for _, mi := range models.AllModels {
		mi.Logger = logger.With("model", mi.Name)
		mi.RequestTimeout = cfg.ModelRequestTimeout
		allModels = append(allModels, mi)
	}
	apikeys.Load(allModels)

	// Log warnings for missing keys
	for _, mi := range allModels {
		if mi.APIKey == "" {
			mi.Logger.Warn("api key missing")
		}
	}
	logger.Info("api keys loaded")

	// Initialize database
	logger.Info("initializing database")
	database, err := db.New("fat.db", logger)
	if err != nil {
		logger.Error("failed to initialize database", slog.Any("error", err))
		panic(fmt.Errorf("failed to initialize database: %w", err))
	}
	defer database.Close()
	logger.Info("database initialized")

	// Start background archiver for answers/ directory
	archiver.StartBackgroundArchiver(logger)

	// Create and run server with embedded static files
	srv := server.New(logger, cfg, database, web.Static)
	if err := srv.Run(); err != nil {
		logger.Error("server exited with error", slog.Any("error", err))
	}
}
