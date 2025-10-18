package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/meedamian/fat/internal/config"
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

	activeModels := make([]*types.ModelInfo, 0, len(models.ModelMap))
	for _, mi := range models.ModelMap {
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

	appLogger.Info("starting server", slog.String("addr", appConfig.ServerAddress))
	if err := r.Run(appConfig.ServerAddress); err != nil {
		appLogger.Error("server exited with error", slog.Any("error", err))
	}
}

func broadcastMessage(message map[string]interface{}) {
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

	defer func() {
		clientsMutex.Lock()
		delete(clients, conn)
		clientsMutex.Unlock()
		conn.Close()
	}()

	ctx := context.Background()

	// Filter models - include all by default for web version
	activeModels := []*types.ModelInfo{}
	for _, mi := range models.ModelMap {
		activeModels = append(activeModels, mi)
	}

	// Check keys for active models
	for _, mi := range activeModels {
		if mi.APIKey == "" {
			appLogger.Warn("api key missing for model", slog.String("model", mi.Name))
		}
	}

	for {
		var msg map[string]interface{}
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

func handleQuestionWS(conn *websocket.Conn, ctx context.Context, activeModels []*types.ModelInfo, msg map[string]interface{}) {
	question, ok := msg["question"].(string)
	if !ok || question == "" {
		conn.WriteJSON(map[string]interface{}{
			"type":  "error",
			"error": "Question is required",
		})
		return
	}

	roundsFloat, ok := msg["rounds"].(float64)
	rounds := int(roundsFloat)
	if !ok || rounds < 1 || rounds > 10 {
		rounds = 3 // Default to 3 rounds
	}

	questionTS := time.Now().Unix()

	// Send loading messages
	for _, mi := range activeModels {
		broadcastMessage(map[string]interface{}{
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

	// Clear previous responses and send round start
	broadcastMessage(map[string]interface{}{
		"type":       "clear",
		"request_id": requestID,
	})

	// Initialize conversation state
	replies := make(map[string]string)      // agent -> latest reply
	discussion := make(map[string][]string) // fromAgent -> list of messages

	for round := 0; round < numRounds; round++ {
		logger.Info("starting round", slog.Int("round", round+1))
		
		broadcastMessage(map[string]interface{}{
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
				
				broadcastMessage(map[string]interface{}{
					"type":       "error",
					"model":      result.modelID,
					"round":      round + 1,
					"error":      result.err.Error(),
					"request_id": requestID,
				})
			} else {
				// Update conversation state
				replies[result.modelID] = result.reply.Answer
				
				// Initialize discussion entry if needed and append messages
				if _, exists := discussion[result.modelID]; !exists {
					discussion[result.modelID] = []string{}
				}
				for targetAgent, message := range result.reply.Discussion {
					discussion[result.modelID] = append(discussion[result.modelID], fmt.Sprintf("To %s: %s", targetAgent, message))
				}

				broadcastMessage(map[string]interface{}{
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
	broadcastMessage(map[string]interface{}{
		"type":       "ranking_start",
		"request_id": requestID,
	})

	winner := rankModels(ctx, requestID, question, replies, activeModels, questionTS, reqMetrics)
	
	reqMetrics.Complete(winner)
	
	logger.Info("question processing complete", slog.Any("metrics", reqMetrics.Summary()))

	broadcastMessage(map[string]interface{}{
		"type":       "winner",
		"model":      winner,
		"answer":     replies[winner],
		"request_id": requestID,
		"metrics":    reqMetrics.Summary(),
	})
}

type callResult struct {
	modelID string
	reply   types.Reply
	err     error
}

func parallelCall(ctx context.Context, requestID, question string, replies map[string]string, discussion map[string][]string, activeModels []*types.ModelInfo, round int, numRounds int, questionTS int64, reqMetrics *metrics.RequestMetrics) <-chan callResult {
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
				timeout = 30 * time.Second
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
			utils.Log(questionTS, fmt.Sprintf("R%d", round+1), mi.Name, result.Prompt, result.Reply.RawContent)

			results <- callResult{
				modelID: mi.ID,
				reply:   result.Reply,
			}
		}(mi)
	}

	return results
}

func rankModels(ctx context.Context, requestID, question string, replies map[string]string, activeModels []*types.ModelInfo, questionTS int64, reqMetrics *metrics.RequestMetrics) string {
	logger := appLogger.With("request_id", requestID)
	logger.Info("starting ranking phase", slog.Int("num_models", len(activeModels)))
	
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
			prompt := shared.FormatRankingPrompt(mi.Name, question, otherAgents, replies)

			// Create timeout context
			timeout := mi.RequestTimeout
			if timeout == 0 {
				timeout = 30 * time.Second
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
			
			result, err := model.Prompt(callCtx, prompt, meta, make(map[string]string), make(map[string][]string))
			
			duration := time.Since(startTime)
			
			if err != nil {
				mi.Logger.Error("ranking failed", slog.Any("error", err))
				return
			}

			// Parse ranking from response
			ranking := shared.ParseRanking(result.Reply.RawContent)
			
			// Log ranking
			utils.Log(questionTS, "rank", mi.Name, prompt, result.Reply.RawContent)

			// Record metrics
			mm := reqMetrics.ModelMetrics[mi.ID]
			if mm != nil {
				mm.RecordRanking(duration, result.TokIn, result.TokOut)
			}

			mu.Lock()
			rankings[mi.ID] = ranking
			mu.Unlock()

			mi.Logger.Info("ranking completed", slog.Any("ranking", ranking))
		}(mi)
	}

	wg.Wait()

	// Aggregate rankings
	allAgentNames := make([]string, 0, len(activeModels))
	for _, mi := range activeModels {
		allAgentNames = append(allAgentNames, mi.Name)
	}

	winner := shared.AggregateRankings(rankings, allAgentNames)
	
	// Convert winner name back to ID
	for _, mi := range activeModels {
		if mi.Name == winner {
			logger.Info("ranking complete", slog.String("winner", winner))
			return mi.ID
		}
	}

	// Fallback to first model with response
	for _, mi := range activeModels {
		if _, ok := replies[mi.ID]; ok {
			logger.Warn("ranking fallback to first responder", slog.String("model", mi.ID))
			return mi.ID
		}
	}

	// Final fallback
	if len(activeModels) > 0 {
		return activeModels[0].ID
	}
	return ""
}

func loadKeys() {
	envVars := map[string]string{
		"grok-4-fast":      "GROK_KEY",
		"gpt-5-mini":       "GPT_KEY",
		"claude-3.5-haiku": "CLAUDE_KEY",
		"gemini-2.5-flash": "GEMINI_KEY",
	}
	jsonKeys := map[string]string{
		"grok-4-fast":      "grok",
		"gpt-5-mini":       "gpt",
		"claude-3.5-haiku": "claude",
		"gemini-2.5-flash": "gemini",
	}
	// Env
	for _, mi := range models.ModelMap {
		if envVar, ok := envVars[mi.Name]; ok {
			key := os.Getenv(envVar)
			if key != "" {
				mi.APIKey = key
				continue
			}
		}
	}
	// .env
	godotenv.Load()
	for _, mi := range models.ModelMap {
		if envVar, ok := envVars[mi.Name]; ok {
			key := os.Getenv(envVar)
			if key != "" {
				mi.APIKey = key
				continue
			}
		}
	}
	// keys.json
	if file, err := os.Open("keys.json"); err == nil {
		defer file.Close()
		var keys map[string]string
		json.NewDecoder(file).Decode(&keys)
		for _, mi := range models.ModelMap {
			if jsonKey, ok := jsonKeys[mi.Name]; ok {
				if key, ok := keys[jsonKey]; ok {
					mi.APIKey = key
				}
			}
		}
	}
	// Check removed, done in main for active models
}
