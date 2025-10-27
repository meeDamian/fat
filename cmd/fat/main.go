package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/meedamian/fat/internal/config"
	"github.com/meedamian/fat/internal/db"
	"github.com/meedamian/fat/internal/metrics"
	"github.com/meedamian/fat/internal/models"
	"github.com/meedamian/fat/internal/retry"
	"github.com/meedamian/fat/internal/shared"
	"github.com/meedamian/fat/internal/types"
	"github.com/meedamian/fat/internal/utils"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}
	clients      = make(map[*websocket.Conn]bool)
	clientsMutex sync.Mutex
	appLogger    *slog.Logger
	appConfig    config.Config
	appDB        *db.DB
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Errorf("failed to load config: %w", err))
	}
	appConfig = cfg

	logger, err := config.NewLogger(cfg.LogLevel)
	if err != nil {
		panic(fmt.Errorf("failed to create logger: %w", err))
	}
	appLogger = logger

	appLogger.Info("loading API keys")
	loadKeys()
	appLogger.Info("api keys loaded")

	// Initialize database
	appLogger.Info("initializing database")
	database, err := db.New("fat.db", appLogger)
	if err != nil {
		appLogger.Error("failed to initialize database", slog.Any("error", err))
		panic(fmt.Errorf("failed to initialize database: %w", err))
	}
	appDB = database
	defer appDB.Close()
	appLogger.Info("database initialized")

	activeModels := make([]*types.ModelInfo, 0, len(models.AllModels))
	for _, mi := range models.AllModels {
		mi.Logger = appLogger.With("model", mi.Name)
		mi.RequestTimeout = appConfig.ModelRequestTimeout
		activeModels = append(activeModels, mi)
		if mi.APIKey == "" {
			mi.Logger.Warn("api key missing")
		}
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.Static("/static", "./static")
	r.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})
	r.GET("/ws", handleWebSocket)

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
			"uptime": time.Since(time.Now()).String(),
		})
	})

	// Stats endpoint
	r.GET("/stats", func(c *gin.Context) {
		ctx := c.Request.Context()

		// Get all model stats
		modelStats, err := appDB.GetAllModelStats(ctx)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		// Get recent requests
		recentRequests, err := appDB.GetRecentRequests(ctx, 10)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"model_stats":     modelStats,
			"recent_requests": recentRequests,
		})
	})

	appLogger.Info("starting server", slog.String("addr", appConfig.ServerAddress))
	if err := r.Run(appConfig.ServerAddress); err != nil {
		appLogger.Error("server exited with error", slog.Any("error", err))
	}
}

// normalizeAgentName converts any agent name variant (ID, full name, or display name) to model ID
func normalizeAgentName(agentName string, activeModels []*types.ModelInfo) string {
	agentName = strings.TrimSpace(agentName)
	agentName = strings.ToLower(agentName)

	for _, mi := range activeModels {
		// Check if it matches the ID
		if strings.ToLower(mi.ID) == agentName {
			return mi.ID
		}
		// Check if it matches the full name
		if strings.ToLower(mi.Name) == agentName {
			return mi.ID
		}
		// Check if it's a partial match (e.g., "grok" matches "grok-4-fast")
		if strings.Contains(strings.ToLower(mi.Name), agentName) {
			return mi.ID
		}
		if strings.Contains(strings.ToLower(mi.ID), agentName) {
			return mi.ID
		}
	}

	return ""
}

func broadcastMessage(message map[string]any) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	messageBytes, _ := json.Marshal(message)

	for client := range clients {
		if err := client.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
			appLogger.Warn("websocket write failed", slog.Any("error", err))
			client.Close()
			delete(clients, client)
		}
	}
}

func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		appLogger.Error("websocket upgrade failed", slog.Any("error", err))
		return
	}

	clientsMutex.Lock()
	clients[conn] = true
	clientsMutex.Unlock()

	// Create cancellable context that will be cancelled when connection closes
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Cancel context when WebSocket closes

	defer func() {
		clientsMutex.Lock()
		delete(clients, conn)
		clientsMutex.Unlock()
		conn.Close()
	}()

	// Filter models - include all by default for web version
	activeModels := []*types.ModelInfo{}
	for _, mi := range models.AllModels {
		activeModels = append(activeModels, mi)
	}

	// Check keys for active models
	for _, mi := range activeModels {
		if mi.APIKey == "" {
			appLogger.Warn("api key missing for model", slog.String("model", mi.Name))
		}
	}

	for {
		var msg map[string]any
		err := conn.ReadJSON(&msg)
		if err != nil {
			appLogger.Debug("websocket read error", slog.Any("error", err))
			break
		}

		msgType, ok := msg["type"].(string)
		if !ok {
			continue
		}

		switch msgType {
		case "question":
			handleQuestionWS(conn, ctx, activeModels, msg)
		}
	}
}

func handleQuestionWS(conn *websocket.Conn, ctx context.Context, activeModels []*types.ModelInfo, msg map[string]any) {
	question, ok := msg["question"].(string)
	if !ok || question == "" {
		conn.WriteJSON(map[string]any{
			"type":  "error",
			"error": "Question is required",
		})
		return
	}

	roundsFloat, ok := msg["rounds"].(float64)
	rounds := int(roundsFloat)
	if !ok || rounds < 3 || rounds > 10 {
		rounds = 3 // Default to 3 rounds
	}

	questionTS := time.Now().Unix()

	// Send loading messages
	for _, mi := range activeModels {
		broadcastMessage(map[string]any{
			"type":  "loading",
			"model": mi.ID,
		})
	}

	// Process question in background
	go func() {
		processQuestion(ctx, question, rounds, activeModels, questionTS)
	}()
}

func processQuestion(ctx context.Context, question string, numRounds int, activeModels []*types.ModelInfo, questionTS int64) {
	// Generate request ID
	requestID := uuid.New().String()
	logger := appLogger.With("request_id", requestID)

	// Initialize metrics
	reqMetrics := metrics.NewRequestMetrics(requestID, question, numRounds, len(activeModels))
	for _, mi := range activeModels {
		reqMetrics.AddModelMetrics(mi.ID)
	}

	logger.Info("starting question processing",
		slog.String("question", question),
		slog.Int("rounds", numRounds),
		slog.Int("models", len(activeModels)))

	// Check for cancellation and create marker file if cancelled
	defer func() {
		if ctx.Err() == context.Canceled {
			logger.Info("request cancelled, creating marker file")
			if err := utils.LogCancellation(questionTS); err != nil {
				logger.Warn("failed to create cancellation marker", slog.Any("error", err))
			}
		}
	}()

	// Clear previous responses and send round start
	broadcastMessage(map[string]any{
		"type":       "clear",
		"request_id": requestID,
	})

	// Initialize conversation state
	replies := make(map[string]types.Reply) // agent -> latest reply with answer and rationale
	// discussion[agentA][agentB] = conversation thread between A and B
	discussion := make(map[string]map[string][]types.DiscussionMessage)

	for round := 0; round < numRounds; round++ {
		logger.Info("starting round", slog.Int("round", round+1))

		broadcastMessage(map[string]any{
			"type":       "round_start",
			"round":      round + 1,
			"total":      numRounds,
			"request_id": requestID,
		})

		results := parallelCall(ctx, requestID, question, replies, discussion, activeModels, round, numRounds, questionTS, reqMetrics)

		// Wait for all models to complete this round
		for range activeModels {
			result := <-results
			if result.err != nil {
				logger.Error("model error",
					slog.String("model", result.modelID),
					slog.Int("round", round+1),
					slog.Any("error", result.err))

				broadcastMessage(map[string]any{
					"type":       "error",
					"model":      result.modelID,
					"round":      round + 1,
					"error":      result.err.Error(),
					"request_id": requestID,
				})
			} else {
				// Update conversation state
				replies[result.modelID] = result.reply

				// Store discussion messages as conversation threads
				// Each message creates a bidirectional thread between sender and recipient
				for targetAgent, message := range result.reply.Discussion {
					// Normalize target agent name to model ID
					targetID := normalizeAgentName(targetAgent, activeModels)
					if targetID == "" {
						logger.Warn("could not normalize agent name",
							slog.String("agent", targetAgent),
							slog.String("from", result.modelID))
						continue
					}

					// Initialize discussion maps if needed
					if _, exists := discussion[result.modelID]; !exists {
						discussion[result.modelID] = make(map[string][]types.DiscussionMessage)
					}
					if _, exists := discussion[targetID]; !exists {
						discussion[targetID] = make(map[string][]types.DiscussionMessage)
					}

					// Add message to both sender's and recipient's conversation threads
					msg := types.DiscussionMessage{
						From:    result.modelID,
						Message: message,
						Round:   round + 1,
					}
					discussion[result.modelID][targetID] = append(discussion[result.modelID][targetID], msg)
					discussion[targetID][result.modelID] = append(discussion[targetID][result.modelID], msg)
				}

				broadcastMessage(map[string]any{
					"type":       "response",
					"model":      result.modelID,
					"round":      round + 1,
					"response":   result.reply.Answer,
					"request_id": requestID,
				})
			}
		}
	}

	// Ranking phase
	logger.Info("starting ranking phase")
	broadcastMessage(map[string]any{
		"type":       "ranking_start",
		"request_id": requestID,
	})

	winnerID, runnerUpID := rankModels(ctx, requestID, question, replies, activeModels, questionTS, reqMetrics)

	reqMetrics.Complete(winnerID)

	logger.Info("question processing complete", slog.Any("metrics", reqMetrics.Summary()))

	// Save to database
	if err := saveToDatabase(ctx, reqMetrics, question, winnerID); err != nil {
		logger.Error("failed to save to database", slog.Any("error", err))
	}

	broadcastMessage(map[string]any{
		"type":       "winner",
		"model":      winnerID,
		"runner_up":  runnerUpID,
		"answer":     replies[winnerID],
		"request_id": requestID,
		"metrics":    reqMetrics.Summary(),
	})
}

type callResult struct {
	modelID string
	reply   types.Reply
	err     error
}

func parallelCall(ctx context.Context, requestID, question string, replies map[string]types.Reply, discussion map[string]map[string][]types.DiscussionMessage, activeModels []*types.ModelInfo, round int, numRounds int, questionTS int64, reqMetrics *metrics.RequestMetrics) <-chan callResult {
	results := make(chan callResult, len(activeModels))

	for _, mi := range activeModels {
		go func(mi *types.ModelInfo) {
			defer func() {
				if r := recover(); r != nil {
					results <- callResult{modelID: mi.ID, err: fmt.Errorf("panic: %v", r)}
				}
			}()

			startTime := time.Now()

			// Calculate other agents (all active models except this one)
			otherAgents := make([]string, 0, len(activeModels)-1)
			for _, m := range activeModels {
				if m.ID != mi.ID {
					otherAgents = append(otherAgents, m.Name)
				}
			}

			meta := types.Meta{
				Round:       round + 1,
				TotalRounds: numRounds,
				OtherAgents: otherAgents,
			}

			// Create timeout context for this model call
			timeout := mi.RequestTimeout
			if timeout == 0 {
				timeout = 60 * time.Second
			}
			callCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			model := models.NewModel(mi)

			// Retry configuration
			retryCfg := retry.DefaultConfig()
			var result types.ModelResult
			var err error

			// Execute with retry
			retryErr := retry.Do(callCtx, retryCfg, func() error {
				result, err = model.Prompt(callCtx, question, meta, replies, discussion)
				if err != nil && retry.IsRetryable(err) {
					mi.Logger.Warn("retrying after error", slog.Any("error", err))
					return err
				}
				return err
			})

			duration := time.Since(startTime)

			if retryErr != nil {
				mi.Logger.Error("model prompt failed after retries",
					slog.Int("round", round+1),
					slog.Any("error", retryErr))

				// Record metrics
				mm := reqMetrics.ModelMetrics[mi.ID]
				if mm != nil {
					mm.RecordRound(round+1, duration, 0, 0, retryErr)
				}

				results <- callResult{modelID: mi.ID, err: fmt.Errorf("model %s: %w", mi.Name, retryErr)}
				return
			}

			// Record metrics
			mm := reqMetrics.ModelMetrics[mi.ID]
			if mm != nil {
				mm.RecordRound(round+1, duration, result.TokIn, result.TokOut, nil)
			}

			// Log the conversation
			if err := utils.Log(questionTS, fmt.Sprintf("R%d", round+1), mi.Name, result.Prompt, result.Reply.RawContent); err != nil {
				mi.Logger.Warn("failed to log conversation", slog.Any("error", err))
			}

			results <- callResult{
				modelID: mi.ID,
				reply:   result.Reply,
			}
		}(mi)
	}

	return results
}

func rankModels(ctx context.Context, requestID, question string, replies map[string]types.Reply, activeModels []*types.ModelInfo, questionTS int64, reqMetrics *metrics.RequestMetrics) (string, string) {
	logger := appLogger.With("request_id", requestID)
	logger.Info("starting ranking phase", slog.Int("num_models", len(activeModels)))

	// Remap replies to use full model names as keys (needed for ranking prompt)
	repliesByName := make(map[string]types.Reply)
	for _, mi := range activeModels {
		if reply, ok := replies[mi.ID]; ok {
			repliesByName[mi.Name] = reply
		}
	}

	// Collect rankings from all models
	rankings := make(map[string][]string)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, mi := range activeModels {
		wg.Add(1)
		go func(mi *types.ModelInfo) {
			defer wg.Done()

			startTime := time.Now()

			// Calculate other agents
			otherAgents := make([]string, 0, len(activeModels)-1)
			for _, m := range activeModels {
				if m.ID != mi.ID {
					otherAgents = append(otherAgents, m.Name)
				}
			}

			// Create ranking prompt
			prompt := shared.FormatRankingPrompt(mi.Name, question, otherAgents, repliesByName)

			// Create timeout context
			timeout := mi.RequestTimeout
			if timeout == 0 {
				timeout = 60 * time.Second
			}
			callCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			// Call model for ranking (using Prompt method for now)
			model := models.NewModel(mi)
			meta := types.Meta{
				Round:       1,
				TotalRounds: 1,
				OtherAgents: otherAgents,
			}

			result, err := model.Prompt(callCtx, prompt, meta, make(map[string]types.Reply), make(map[string]map[string][]types.DiscussionMessage))

			duration := time.Since(startTime)

			if err != nil {
				mi.Logger.Error("ranking failed", slog.Any("error", err))
				return
			}

			// Parse ranking from response
			ranking := shared.ParseRanking(result.Reply.RawContent)

			// Log ranking
			if err := utils.Log(questionTS, "rank", mi.Name, prompt, result.Reply.RawContent); err != nil {
				mi.Logger.Warn("failed to log ranking", slog.Any("error", err))
			}

			// Record metrics
			mm := reqMetrics.ModelMetrics[mi.ID]
			if mm != nil {
				mm.RecordRanking(duration, result.TokIn, result.TokOut)
			}

			// Save ranking to database
			if len(ranking) > 0 {
				rankedModelsJSON, _ := json.Marshal(ranking)
				// Calculate cost: (tokens_in * rate_in + tokens_out * rate_out) / 1M
				rankingCost := (float64(result.TokIn)*mi.Rates.In + float64(result.TokOut)*mi.Rates.Out) / 1_000_000
				rankingRecord := db.Ranking{
					RequestID:    requestID,
					RankerModel:  mi.Name,
					RankedModels: string(rankedModelsJSON),
					DurationMs:   duration.Milliseconds(),
					TokensIn:     int64(result.TokIn),
					TokensOut:    int64(result.TokOut),
					Cost:         rankingCost,
				}
				if err := appDB.SaveRanking(ctx, rankingRecord); err != nil {
					mi.Logger.Warn("failed to save ranking to database", slog.Any("error", err))
				}
			}

			mu.Lock()
			if len(ranking) == 0 {
				mi.Logger.Warn("model failed to provide ranking - likely provided answer instead")
			} else {
				rankings[mi.ID] = ranking
			}
			mu.Unlock()

			mi.Logger.Info("ranking completed", slog.Any("ranking", ranking), slog.Int("count", len(ranking)))
		}(mi)
	}

	wg.Wait()

	// Aggregate rankings
	allAgentNames := make([]string, 0, len(activeModels))
	for _, mi := range activeModels {
		allAgentNames = append(allAgentNames, mi.Name)
	}

	// Log how many valid rankings we got
	logger.Info("aggregating rankings",
		slog.Int("valid_rankings", len(rankings)),
		slog.Int("total_models", len(activeModels)))

	winner, runnerUp := shared.AggregateRankings(rankings, allAgentNames)

	// Convert winner name back to ID
	winnerID := ""
	runnerUpID := ""
	for _, mi := range activeModels {
		if mi.Name == winner {
			winnerID = mi.ID
		}
		if mi.Name == runnerUp {
			runnerUpID = mi.ID
		}
	}

	if winnerID != "" {
		logger.Info("ranking complete",
			slog.String("winner", winner),
			slog.String("runner_up", runnerUp))
		return winnerID, runnerUpID
	}

	// Fallback to first model with response
	for _, mi := range activeModels {
		if _, ok := replies[mi.ID]; ok {
			logger.Warn("ranking fallback to first responder", slog.String("model", mi.ID))
			return mi.ID, ""
		}
	}

	// Final fallback
	if len(activeModels) > 0 {
		return activeModels[0].ID, ""
	}
	return "", ""
}

// saveToDatabase persists request metrics to SQLite
func saveToDatabase(ctx context.Context, reqMetrics *metrics.RequestMetrics, question, winner string) error {
	summary := reqMetrics.Summary()

	// Calculate total cost based on model rates
	totalCost := 0.0
	for modelID, mm := range reqMetrics.ModelMetrics {
		// Get model info for rates
		var modelInfo *types.ModelInfo
		for _, mi := range models.AllModels {
			if mi.ID == modelID {
				modelInfo = mi
				break
			}
		}

		if modelInfo != nil {
			// Calculate cost: (tokens_in * rate_in + tokens_out * rate_out) / 1M
			cost := (float64(mm.TotalTokens.Input)*modelInfo.Rates.In + float64(mm.TotalTokens.Output)*modelInfo.Rates.Out) / 1_000_000
			totalCost += cost
		}
	}

	// Save main request record
	req := db.Request{
		ID:              reqMetrics.RequestID,
		Question:        question,
		NumRounds:       reqMetrics.NumRounds,
		NumModels:       reqMetrics.NumModels,
		WinnerModel:     winner,
		TotalDurationMs: reqMetrics.Duration().Milliseconds(),
		TotalTokensIn:   summary["total_tokens_in"].(int64),
		TotalTokensOut:  summary["total_tokens_out"].(int64),
		TotalCost:       totalCost,
		ErrorCount:      summary["error_count"].(int),
	}

	if err := appDB.SaveRequest(ctx, req); err != nil {
		return fmt.Errorf("failed to save request: %w", err)
	}

	// Save individual model rounds
	for modelID, mm := range reqMetrics.ModelMetrics {
		var modelInfo *types.ModelInfo
		for _, mi := range models.AllModels {
			if mi.ID == modelID {
				modelInfo = mi
				break
			}
		}

		if modelInfo == nil {
			continue
		}

		for _, roundMetric := range mm.RoundMetrics {
			cost := (float64(roundMetric.Tokens.Input)*modelInfo.Rates.In + float64(roundMetric.Tokens.Output)*modelInfo.Rates.Out) / 1_000_000

			mr := db.ModelRound{
				RequestID:  reqMetrics.RequestID,
				ModelID:    modelID,
				ModelName:  modelInfo.Name,
				Round:      roundMetric.Round,
				DurationMs: roundMetric.Duration.Milliseconds(),
				TokensIn:   roundMetric.Tokens.Input,
				TokensOut:  roundMetric.Tokens.Output,
				Cost:       cost,
				Error:      roundMetric.Error,
			}

			if err := appDB.SaveModelRound(ctx, mr); err != nil {
				appLogger.Warn("failed to save model round",
					slog.String("model", modelID),
					slog.Int("round", roundMetric.Round),
					slog.Any("error", err))
			}
		}

		// Update model stats
		won := (modelID == winner)
		avgResponseTime := int64(0)
		if len(mm.RoundMetrics) > 0 {
			totalTime := int64(0)
			for _, rm := range mm.RoundMetrics {
				totalTime += rm.Duration.Milliseconds()
			}
			avgResponseTime = totalTime / int64(len(mm.RoundMetrics))
		}

		modelCost := (float64(mm.TotalTokens.Input)*modelInfo.Rates.In + float64(mm.TotalTokens.Output)*modelInfo.Rates.Out) / 1_000_000

		if err := appDB.UpdateModelStats(ctx, modelID, modelInfo.Name, won,
			mm.TotalTokens.Input, mm.TotalTokens.Output, modelCost, avgResponseTime); err != nil {
			appLogger.Warn("failed to update model stats",
				slog.String("model", modelID),
				slog.Any("error", err))
		}
	}

	return nil
}

func loadKeys() {
	// Map model family IDs to their environment variable names
	// This is the only mapping needed - works with any model variant
	familyEnvVars := map[string]string{
		models.Grok:   "GROK_KEY",
		models.GPT:    "GPT_KEY",
		models.Claude: "CLAUDE_KEY",
		models.Gemini: "GEMINI_KEY",
	}

	// Try environment variables first
	for _, mi := range models.AllModels {
		if envVar, ok := familyEnvVars[mi.ID]; ok {
			key := os.Getenv(envVar)
			if key != "" {
				mi.APIKey = key
				continue
			}
		}
	}

	// Try .env file
	godotenv.Load()
	for _, mi := range models.AllModels {
		if mi.APIKey != "" {
			continue // Already loaded from env
		}
		if envVar, ok := familyEnvVars[mi.ID]; ok {
			key := os.Getenv(envVar)
			if key != "" {
				mi.APIKey = key
				continue
			}
		}
	}

	// Try keys.json (uses family ID as key)
	if file, err := os.Open("keys.json"); err == nil {
		defer file.Close()
		var keys map[string]string
		json.NewDecoder(file).Decode(&keys)
		for _, mi := range models.AllModels {
			if mi.APIKey != "" {
				continue // Already loaded
			}
			if key, ok := keys[mi.ID]; ok {
				mi.APIKey = key
			}
		}
	}
}
