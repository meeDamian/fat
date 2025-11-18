package ranking

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/meedamian/fat/internal/db"
	"github.com/meedamian/fat/internal/metrics"
	"github.com/meedamian/fat/internal/models"
	"github.com/meedamian/fat/internal/shared"
	"github.com/meedamian/fat/internal/types"
	"github.com/meedamian/fat/internal/utils"
)

// RankModels executes the ranking phase where all models rank each other's responses
// Returns gold, silver, and bronze winner IDs (can have multiple winners for ties)
func RankModels(
	ctx context.Context,
	requestID string,
	question string,
	replies map[string]types.Reply,
	activeModels []*types.ModelInfo,
	questionTS int64,
	reqMetrics *metrics.RequestMetrics,
	database *db.DB,
	logger *slog.Logger,
) ([]string, []string, []string) {
	logger = logger.With("request_id", requestID)
	logger.Info("starting ranking phase", slog.Int("num_models", len(activeModels)))

	// Remap replies to use full model names as keys (needed for ranking prompt)
	repliesByName := make(map[string]types.Reply)
	for _, mi := range activeModels {
		if reply, ok := replies[mi.ID]; ok {
			repliesByName[mi.Name] = reply
		}
	}

	// Calculate costs for each model
	costsByName := make(map[string]float64)
	for _, mi := range activeModels {
		mm := reqMetrics.ModelMetrics[mi.ID]
		if mm != nil {
			rate := getRateForModel(mi)
			cost := (float64(mm.TotalTokens.Input)*rate.In + float64(mm.TotalTokens.Output)*rate.Out) / 1_000_000
			costsByName[mi.Name] = cost
		}
	}

	// Create shared anonymization map for all models
	allAgentNames := make([]string, 0, len(activeModels))
	for _, mi := range activeModels {
		allAgentNames = append(allAgentNames, mi.Name)
	}
	anonMap := shared.CreateAnonymizationMap(allAgentNames)

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

			// Create ranking prompt with shared anonymization map and costs
			prompt := shared.FormatRankingPrompt(mi.Name, question, otherAgents, repliesByName, anonMap, costsByName)

			// Create timeout context
			timeout := mi.RequestTimeout
			if timeout == 0 {
				timeout = 60 * time.Second
			}
			callCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			// Call model for ranking
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
			ranking := shared.ParseRanking(result.Reply.RawContent, prompt)

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
				rate := getRateForModel(mi)
				rankingCost := (float64(result.TokIn)*rate.In + float64(result.TokOut)*rate.Out) / 1_000_000
				rankingRecord := db.Ranking{
					RequestID:    requestID,
					RankerModel:  mi.Name,
					RankedModels: string(rankedModelsJSON),
					DurationMs:   duration.Milliseconds(),
					TokensIn:     int64(result.TokIn),
					TokensOut:    int64(result.TokOut),
					Cost:         rankingCost,
				}
				if err := database.SaveRanking(ctx, rankingRecord); err != nil {
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

	// Log how many valid rankings we got
	logger.Info("aggregating rankings",
		slog.Int("valid_rankings", len(rankings)),
		slog.Int("total_models", len(activeModels)))

	goldNames, silverNames, bronzeNames := shared.AggregateRankings(rankings, allAgentNames)

	// Convert names back to IDs
	goldIDs := make([]string, 0, len(goldNames))
	silverIDs := make([]string, 0, len(silverNames))
	bronzeIDs := make([]string, 0, len(bronzeNames))

	for _, mi := range activeModels {
		for _, name := range goldNames {
			if mi.Name == name {
				goldIDs = append(goldIDs, mi.ID)
			}
		}
		for _, name := range silverNames {
			if mi.Name == name {
				silverIDs = append(silverIDs, mi.ID)
			}
		}
		for _, name := range bronzeNames {
			if mi.Name == name {
				bronzeIDs = append(bronzeIDs, mi.ID)
			}
		}
	}

	if len(goldIDs) > 0 {
		logger.Info("ranking complete",
			slog.Any("gold", goldNames),
			slog.Any("silver", silverNames),
			slog.Any("bronze", bronzeNames))
		return goldIDs, silverIDs, bronzeIDs
	}

	// Fallback to first model with response
	for _, mi := range activeModels {
		if _, ok := replies[mi.ID]; ok {
			logger.Warn("ranking fallback to first responder", slog.String("model", mi.ID))
			return []string{mi.ID}, []string{}, []string{}
		}
	}

	// Final fallback
	logger.Warn("no ranking winner, returning first active model")
	return []string{activeModels[0].ID}, []string{}, []string{}
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
