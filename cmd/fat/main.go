package main

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

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
	grokFlag      = flag.Bool("grok", false, "Include Grok model")
	gptFlag       = flag.Bool("gpt", false, "Include GPT model")
	claudeFlag    = flag.Bool("claude", false, "Include Claude model")
	geminiFlag    = flag.Bool("gemini", false, "Include Gemini model")
	noGrokFlag    = flag.Bool("no-grok", false, "Exclude Grok model")
	noGptFlag     = flag.Bool("no-gpt", false, "Exclude GPT model")
	noClaudeFlag  = flag.Bool("no-claude", false, "Exclude Claude model")
	noGeminiFlag  = flag.Bool("no-gemini", false, "Exclude Gemini model")
)

func init() {
}

func main() {
	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("Question required")
	}
	question := strings.Join(args, " ")

	// Generate base64 timestamp for file prefixes
	ts := time.Now().Unix()
	base64TS := base64.StdEncoding.EncodeToString([]byte(strconv.FormatInt(ts, 10)))

	// Set in utils or something, but since utils.Log is called, modify utils.Log to take prefix.

	// For now, global
	utils.SetBase64TS(base64TS)

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

	// Check keys for active models
	for _, mi := range activeModels {
		if mi.APIKey == "" {
			log.Fatalf("API key for %s missing", mi.Name)
		}
	}

	// Determine rounds
	numRounds := *roundsFlag
	if numRounds == -1 {
		if *verbose {
			fmt.Printf("Estimating rounds for question: %s\n", question)
		}
		if len(activeModels) == 1 {
			numRounds = 1
		} else {
			numRounds = estimateRounds(question, activeModels)
		}
	}
	if *verbose {
		fmt.Printf("Decided on %d rounds\n", numRounds)
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
		if *verbose {
			fmt.Printf("Calling model %s with prompt: %s\n", mi.Name, prompt)
		}
		resp, tokIn, tokOut, err := models.CallModel(ctx, mi, prompt, nil)
		if err != nil {
			log.Fatal(err)
		}
		if *verbose {
			fmt.Printf("Response from %s: %s (tokens in: %d, out: %d)\n", mi.Name, resp.Refined, tokIn, tokOut)
		}
		utils.Log(mi.Name, prompt, resp.Refined)
		fmt.Println(resp.Refined)
	} else {
		// Multi mode
		history := make(types.History)
		for round := 0; round < numRounds; round++ {
			if *verbose {
				fmt.Printf("Starting round %d/%d\n", round+1, numRounds)
			}
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
			if *verbose {
				fmt.Printf("Round %d/%d completed\n", round+1, numRounds)
			}
		}
		// Rank
		ranks := rankModels(ctx, question, history, activeModels)
		winner := selectWinner(ranks)
		fmt.Printf("Model %s was decided to be the best.\n\n%s\n", models.ModelMap[winner].Name, history[winner][len(history[winner])-1].Refined)
		if *verbose {
			for id, score := range ranks {
				fmt.Printf("Model %s: %d\n", models.ModelMap[id].Name, score)
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
	// Check removed, done in main for active models
}

func filterModels() []*types.ModelInfo {
	var active []*types.ModelInfo
	includes := []string{}
	if *grokFlag { includes = append(includes, "grok") }
	if *gptFlag { includes = append(includes, "gpt") }
	if *claudeFlag { includes = append(includes, "claude") }
	if *geminiFlag { includes = append(includes, "gemini") }
	excludes := []string{}
	if *noGrokFlag { excludes = append(excludes, "grok") }
	if *noGptFlag { excludes = append(excludes, "gpt") }
	if *noClaudeFlag { excludes = append(excludes, "claude") }
	if *noGeminiFlag { excludes = append(excludes, "gemini") }
	for _, mi := range models.ModelMap {
		if len(includes) > 0 && !slices.Contains(includes, mi.ID) {
			continue
		}
		if slices.Contains(excludes, mi.ID) {
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
	grok := models.ModelMap["grok"]
	prompt := prompts.RankPrompt(question, utils.BuildContext(question, history), activeModels)
	if *verbose {
		fmt.Printf("Sending ranking request: %s\n", prompt)
	}
	utils.Log("ranking", prompt, "")
	resp, _, _, err := models.CallModel(ctx, grok, prompt, nil)
	if err != nil {
		return types.Rank{}
	}
	utils.Log("ranking", "", resp.Refined)
	if *verbose {
		fmt.Printf("Ranking response: %s\n", resp.Refined)
	}
	// Parse ranking A > B > C
	ranking := strings.Split(resp.Refined, " > ")
	modelLetters := map[string]string{"grok": "A", "gpt": "B", "claude": "C", "gemini": "D"}
	letterToId := make(map[string]string)
	for id, letter := range modelLetters {
		letterToId[letter] = id
	}
	rank := make(types.Rank)
	for i, letter := range ranking {
		if id, ok := letterToId[strings.TrimSpace(letter)]; ok {
			rank[id] = len(ranking) - i
		}
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
