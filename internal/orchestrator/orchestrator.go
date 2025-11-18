package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/meedamian/fat/internal/db"
	"github.com/meedamian/fat/internal/htmlexport"
	"github.com/meedamian/fat/internal/metrics"
	"github.com/meedamian/fat/internal/models"
	"github.com/meedamian/fat/internal/ranking"
	"github.com/meedamian/fat/internal/retry"
	"github.com/meedamian/fat/internal/types"
	"github.com/meedamian/fat/internal/utils"
)

// Broadcaster is an interface for broadcasting messages to connected clients
type Broadcaster interface {
	Broadcast(message map[string]any)
}

// Orchestrator coordinates the multi-round question processing
type Orchestrator struct {
	logger      *slog.Logger
	database    *db.DB
	broadcaster Broadcaster
	exporter    *htmlexport.Exporter
}

// New creates a new Orchestrator
func New(logger *slog.Logger, database *db.DB, broadcaster Broadcaster, exporter *htmlexport.Exporter) *Orchestrator {
	return &Orchestrator{
		logger:      logger,
		database:    database,
		broadcaster: broadcaster,
		exporter:    exporter,
	}
}

// ProcessQuestion orchestrates the entire question processing workflow
func (o *Orchestrator) ProcessQuestion(
	ctx context.Context,
	question string,
	numRounds int,
	activeModels []*types.ModelInfo,
	questionTS int64,
) {
	// Generate request ID
	requestID := uuid.New().String()
	logger := o.logger.With("request_id", requestID)

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
	o.broadcaster.Broadcast(map[string]any{
		"type":       "clear",
		"request_id": requestID,
	})

	// Initialize conversation state
	replies := make(map[string]types.Reply)
	discussion := make(map[string]map[string][]types.DiscussionMessage)

	// Execute rounds
	for round := 0; round < numRounds; round++ {
		logger.Info("starting round", slog.Int("round", round+1))

		o.broadcaster.Broadcast(map[string]any{
			"type":       "round_start",
			"round":      round + 1,
			"total":      numRounds,
			"request_id": requestID,
		})

		results := o.parallelCall(ctx, requestID, question, replies, discussion, activeModels, round, numRounds, questionTS, reqMetrics)

		// Wait for all models to complete this round
		for range activeModels {
			result := <-results
			if result.err != nil {
				logger.Error("model error",
					slog.String("model", result.modelID),
					slog.Int("round", round+1),
					slog.Any("error", result.err))

				o.broadcaster.Broadcast(map[string]any{
					"type":       "error",
					"model":      result.modelID,
					"round":      round + 1,
					"error":      result.err.Error(),
					"request_id": requestID,
				})
			} else {
				// Update conversation state
				replies[result.modelID] = result.reply

				// Store discussion messages
				for targetAgent, message := range result.reply.Discussion {
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

				o.broadcaster.Broadcast(map[string]any{
					"type":       "response",
					"model":      result.modelID,
					"round":      round + 1,
					"response":   result.reply.Answer,
					"rationale":  result.reply.Rationale,
					"discussion": result.reply.Discussion,
					"tokens_in":  result.tokensIn,
					"tokens_out": result.tokensOut,
					"cost":       result.cost,
					"request_id": requestID,
				})
			}
		}
	}

	// Ranking phase
	logger.Info("starting ranking phase")
	o.broadcaster.Broadcast(map[string]any{
		"type":       "ranking_start",
		"request_id": requestID,
	})

	goldIDs, silverIDs, bronzeIDs, scoresByID := ranking.RankModels(ctx, requestID, question, replies, activeModels, questionTS, reqMetrics, o.database, logger)

	// Use first gold winner for metrics completion and broadcast
	winnerID := ""
	if len(goldIDs) > 0 {
		winnerID = goldIDs[0]
	}
	reqMetrics.Complete(winnerID)

	logger.Info("question processing complete", slog.Any("metrics", reqMetrics.Summary()))

	// Save to database
	if err := o.saveToDatabase(ctx, reqMetrics, question, winnerID); err != nil {
		logger.Error("failed to save to database", slog.Any("error", err))
	}

	// For backwards compatibility, broadcast first gold and first silver
	runnerUpID := ""
	if len(silverIDs) > 0 {
		runnerUpID = silverIDs[0]
	}
	o.broadcaster.Broadcast(map[string]any{
		"type":       "winner",
		"model":      winnerID,
		"runner_up":  runnerUpID,
		"answer":     replies[winnerID],
		"gold":       goldIDs,
		"silver":     silverIDs,
		"bronze":     bronzeIDs,
		"request_id": requestID,
		"metrics":    reqMetrics.Summary(),
	})

	// Export static HTML
	if o.exporter != nil {
		if err := o.exportStaticHTML(ctx, question, questionTS, replies, discussion, goldIDs, silverIDs, bronzeIDs, scoresByID, activeModels, reqMetrics); err != nil {
			logger.Error("failed to export static HTML", slog.Any("error", err))
		}
	}
}

// exportStaticHTML generates and saves a static HTML snapshot
func (o *Orchestrator) exportStaticHTML(
	ctx context.Context,
	question string,
	questionTS int64,
	replies map[string]types.Reply,
	discussion map[string]map[string][]types.DiscussionMessage,
	goldIDs, silverIDs, bronzeIDs []string,
	scoresByID map[string]int,
	activeModels []*types.ModelInfo,
	reqMetrics *metrics.RequestMetrics,
) error {
	// Convert discussions to export format
	var discussions []htmlexport.DiscussionPair
	processed := make(map[string]bool)

	for modelA, partners := range discussion {
		for modelB, messages := range partners {
			// Create a unique pair key to avoid duplicates
			pairKey := modelA + "-" + modelB
			reversePairKey := modelB + "-" + modelA

			if processed[pairKey] || processed[reversePairKey] {
				continue
			}
			processed[pairKey] = true

			if len(messages) == 0 {
				continue
			}

			// Find display names
			var nameA, nameB string
			for _, m := range activeModels {
				if m.ID == modelA {
					nameA = m.ID
				}
				if m.ID == modelB {
					nameB = m.ID
				}
			}

			// Convert messages
			var exportMessages []htmlexport.DiscussionMessage
			for _, msg := range messages {
				var fromName string
				for _, m := range activeModels {
					if m.ID == msg.From {
						fromName = m.ID
						break
					}
				}
				exportMessages = append(exportMessages, htmlexport.DiscussionMessage{
					Meta: fmt.Sprintf("%s • Round %d", fromName, msg.Round),
					Text: msg.Message,
				})
			}

			discussions = append(discussions, htmlexport.DiscussionPair{
				Header:   fmt.Sprintf("%s ⟷ %s", nameA, nameB),
				Messages: exportMessages,
			})
		}
	}

	// Extract round counts from metrics
	roundCounts := make(map[string]int)
	for modelID, modelMetrics := range reqMetrics.ModelMetrics {
		roundCounts[modelID] = len(modelMetrics.RoundMetrics)
	}

	// Calculate costs for each model
	modelCosts := make(map[string]string)
	for _, model := range activeModels {
		if mm, ok := reqMetrics.ModelMetrics[model.ID]; ok {
			rate := getRateForModel(model)
			tokensIn := mm.TotalTokens.Input
			tokensOut := mm.TotalTokens.Output
			cost := (float64(tokensIn) * rate.In / 1_000_000) + (float64(tokensOut) * rate.Out / 1_000_000)
			if cost > 0 {
				modelCosts[model.ID] = fmt.Sprintf("$%.4f", cost)
			}
		}
	}

	// Prepare export data
	exportData := htmlexport.ExportData{
		Question:    question,
		QuestionTS:  questionTS,
		GoldIDs:     goldIDs,
		SilverIDs:   silverIDs,
		BronzeIDs:   bronzeIDs,
		Replies:     replies,
		Models:      activeModels,
		Metrics:     reqMetrics.Summary(),
		RoundCounts: roundCounts,
		ModelCosts:  modelCosts,
		ModelScores: scoresByID,
		Discussions: discussions,
		Timestamp:   time.Now().Format("2006-01-02 15:04:05 MST"),
	}

	return o.exporter.Export(ctx, exportData)
}

type callResult struct {
	modelID   string
	reply     types.Reply
	tokensIn  int64
	tokensOut int64
	cost      float64
	err       error
}

func (o *Orchestrator) parallelCall(
	ctx context.Context,
	requestID string,
	question string,
	replies map[string]types.Reply,
	discussion map[string]map[string][]types.DiscussionMessage,
	activeModels []*types.ModelInfo,
	round int,
	numRounds int,
	questionTS int64,
	reqMetrics *metrics.RequestMetrics,
) <-chan callResult {
	results := make(chan callResult, len(activeModels))

	for _, mi := range activeModels {
		go func(mi *types.ModelInfo) {
			defer func() {
				if r := recover(); r != nil {
					results <- callResult{modelID: mi.ID, err: fmt.Errorf("panic: %v", r)}
				}
			}()

			startTime := time.Now()

			// Calculate other agents
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

			// Create timeout context
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

			// Calculate cost
			rate := getRateForModel(mi)
			cost := (float64(result.TokIn)*rate.In + float64(result.TokOut)*rate.Out) / 1_000_000

			results <- callResult{
				modelID:   mi.ID,
				reply:     result.Reply,
				tokensIn:  result.TokIn,
				tokensOut: result.TokOut,
				cost:      cost,
			}
		}(mi)
	}

	return results
}

// saveToDatabase persists request metrics to SQLite
func (o *Orchestrator) saveToDatabase(ctx context.Context, reqMetrics *metrics.RequestMetrics, question, winner string) error {
	summary := reqMetrics.Summary()

	// Calculate total cost
	totalCost := 0.0
	for modelID, mm := range reqMetrics.ModelMetrics {
		var modelInfo *types.ModelInfo
		for _, mi := range models.AllModels {
			if mi.ID == modelID {
				modelInfo = mi
				break
			}
		}

		if modelInfo != nil {
			rate := getRateForModel(modelInfo)
			cost := (float64(mm.TotalTokens.Input)*rate.In + float64(mm.TotalTokens.Output)*rate.Out) / 1_000_000
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

	if err := o.database.SaveRequest(ctx, req); err != nil {
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

		rate := getRateForModel(modelInfo)
		for _, roundMetric := range mm.RoundMetrics {
			cost := (float64(roundMetric.Tokens.Input)*rate.In + float64(roundMetric.Tokens.Output)*rate.Out) / 1_000_000

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

			if err := o.database.SaveModelRound(ctx, mr); err != nil {
				o.logger.Warn("failed to save model round",
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

		modelCost := (float64(mm.TotalTokens.Input)*rate.In + float64(mm.TotalTokens.Output)*rate.Out) / 1_000_000

		if err := o.database.UpdateModelStats(ctx, modelID, modelInfo.Name, won,
			mm.TotalTokens.Input, mm.TotalTokens.Output, modelCost, avgResponseTime); err != nil {
			o.logger.Warn("failed to update model stats",
				slog.String("model", modelID),
				slog.Any("error", err))
		}
	}

	return nil
}

// normalizeAgentName converts any agent name variant to model ID
func normalizeAgentName(agentName string, activeModels []*types.ModelInfo) string {
	agentName = strings.TrimSpace(agentName)
	agentName = strings.ToLower(agentName)

	for _, mi := range activeModels {
		if strings.ToLower(mi.ID) == agentName {
			return mi.ID
		}
		if strings.ToLower(mi.Name) == agentName {
			return mi.ID
		}
		if strings.Contains(strings.ToLower(mi.Name), agentName) {
			return mi.ID
		}
		if strings.Contains(strings.ToLower(mi.ID), agentName) {
			return mi.ID
		}
	}

	return ""
}

// getRateForModel retrieves the pricing rate for a model by looking up its variant
func getRateForModel(modelInfo *types.ModelInfo) types.Rate {
	family, ok := models.ModelFamilies[modelInfo.ID]
	if !ok {
		return types.Rate{}
	}

	variant, ok := family.Variants[modelInfo.Name]
	if !ok {
		return types.Rate{}
	}

	return variant.Rate
}
