package main

import (
	"bufio"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"sync"

	"github.com/joho/godotenv"
	"github.com/meedamian/fat/internal/constants"
	"github.com/meedamian/fat/internal/models"
	"github.com/meedamian/fat/internal/prompts"
	"github.com/meedamian/fat/internal/types"
	"github.com/meedamian/fat/internal/utils"
)

var (
	roundsFlag    = flag.Int("rounds", -1, "Number of rounds (1-10, -1=auto)")
	fullContext   = flag.Bool("full-context", false, "Use full history")
	verbose       = flag.Bool("verbose", false, "Verbose output")
	budget        = flag.Bool("budget", false, "Estimate and confirm budget")
	modelIncludes []string
	modelExcludes []string
)

func init() {
	flag.Var((*stringSlice)(&modelIncludes), "model", "Include model (A/B/C/D)")
	flag.Var((*stringSlice)(&modelExcludes), "no-model", "Exclude model (A/B/C/D)")
}

type stringSlice []string

func (s *stringSlice) String() string         { return strings.Join(*s, ",") }
func (s *stringSlice) Set(value string) error { *s = append(*s, value); return nil }

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("Question required")
	}
	question := strings.Join(args, " ")

	// Load keys
	loadKeys()

	ctx := context.Background()

	// Load rates
	rates := utils.LoadRates(ctx)

	// Init clients
	models.InitClients(rates)

	// Filter models
	activeModels := filterModels()
	if len(activeModels) == 0 {
		log.Fatal("At least one model required")
	}

	// Determine rounds
	numRounds := *roundsFlag
	if numRounds == -1 {
		numRounds = estimateRounds(question, activeModels)
	}
	if numRounds > 1 && len(activeModels) == 1 {
		log.Fatal("Rounds >1 require multiple models")
	}

	if *budget {
		est := estimateBudget(question, activeModels, numRounds)
		fmt.Printf("Estimated cost: $%.4f\nConfirm? (y/n): ", est)
		reader := bufio.NewReader(os.Stdin)
		resp, _ := reader.ReadString('\n')
		if strings.TrimSpace(resp) != "y" {
			return
		}
	}

	if numRounds == 1 || len(activeModels) == 1 {
		// Single mode
		mi := activeModels[0]
		prompt := prompts.InitialPrompt(question)
		resp, _, _, err := models.CallModel(ctx, mi, prompt, nil)
		if err != nil {
			log.Fatal(err)
		}
		utils.Log(mi.Name, prompt, resp.Refined)
		fmt.Println(resp.Refined)
	} else {
		// Multi mode
		history := make(types.History)
		for round := 0; round < numRounds; round++ {
			results := parallelCall(ctx, question, history, activeModels)
			for _, res := range results {
				if res.Err != nil {
					if *verbose {
						log.Printf("Model %s error: %v", res.ID, res.Err)
					}
					continue
				}
				history[res.ID] = append(history[res.ID], res.Resp)
				prompt := prompts.RefinePrompt(question, utils.BuildContext(question, history))
				utils.Log(models.ModelMap[res.ID].Name, prompt, res.Resp.Refined)
			}
		}
		// Rank
		ranks := rankModels(ctx, question, history, activeModels)
		winner := selectWinner(ranks)
		fmt.Printf("Model %s was decided to be the best.\n\n%s\n", winner, history[winner][len(history[winner])-1].Refined)
		if *verbose {
			for id, score := range ranks {
				fmt.Printf("Model %s: %d\n", id, score)
			}
		}
	}
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
	// Check
	for _, mi := range models.ModelMap {
		if mi.APIKey == "" {
			log.Fatalf("API key for %s missing", mi.Name)
		}
	}
}

func filterModels() []*types.ModelInfo {
	var active []*types.ModelInfo
	for _, mi := range models.ModelMap {
		if len(modelIncludes) > 0 && !slices.Contains(modelIncludes, mi.ID) {
			continue
		}
		if slices.Contains(modelExcludes, mi.ID) {
			continue
		}
		active = append(active, mi)
	}
	return active
}

func estimateRounds(question string, models []*types.ModelInfo) int {
	// Stub: return 3
	return 3
}

func estimateBudget(question string, activeModels []*types.ModelInfo, rounds int) float64 {
	estTokIn := utils.EstTokens(question) * int64(rounds)
	estTokOut := constants.EstTokOutPerRound * int64(rounds)
	total := 0.0
	for _, mi := range activeModels {
		total += models.CostForToks(mi, estTokIn, estTokOut)
	}
	return total
}

func parallelCall(ctx context.Context, question string, history types.History, activeModels []*types.ModelInfo) map[string]types.RoundRes {
	results := make(map[string]types.RoundRes)
	var wg sync.WaitGroup
	ch := make(chan types.RoundRes, len(activeModels))
	for _, mi := range activeModels {
		wg.Add(1)
		go func(mi *types.ModelInfo) {
			defer wg.Done()
			context := utils.BuildContext(question, history)
			if !*fullContext {
				context = utils.CapContext(context, mi.MaxTok)
			}
			prompt := prompts.RefinePrompt(question, context)
			resp, _, _, err := models.CallModel(ctx, mi, prompt, nil) // history as messages?
			ch <- types.RoundRes{ID: mi.ID, Resp: resp, Err: err}
		}(mi)
	}
	wg.Wait()
	close(ch)
	for res := range ch {
		results[res.ID] = res
	}
	return results
}

func rankModels(ctx context.Context, question string, history types.History, activeModels []*types.ModelInfo) types.Rank {
	// Use one model to rank, e.g., Grok
	grok := models.ModelMap["A"]
	options := []string{}
	for id := range history {
		last := history[id][len(history[id])-1]
		options = append(options, last.Refined)
	}
	prompt := prompts.RankPrompt(question, utils.BuildContext(question, history), options)
	resp, _, _, err := models.CallModel(ctx, grok, prompt, nil)
	if err != nil {
		return types.Rank{}
	}
	// Parse ranking A > B > C
	ranking := strings.Split(resp.Refined, " > ")
	rank := make(types.Rank)
	for i, id := range ranking {
		rank[strings.TrimSpace(id)] = len(ranking) - i
	}
	return rank
}

func selectWinner(ranks types.Rank) string {
	maxScore := 0
	winner := ""
	for id, score := range ranks {
		if score > maxScore {
			maxScore = score
			winner = id
		}
	}
	return winner
}
