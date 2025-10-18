package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/meedamian/fat/internal/models"
	"github.com/meedamian/fat/internal/prompts"
	"github.com/meedamian/fat/internal/types"
	"github.com/meedamian/fat/internal/utils"
)

var (
	roundsFlag   = flag.Int("rounds", -1, "Number of rounds (1-10, -1=auto)")
	fullContext  = flag.Bool("full-context", false, "Use full history")
	verbose      = flag.Bool("verbose", false, "Verbose output")
	budget       = flag.Bool("budget", false, "Estimate and confirm budget")
	grokFlag     = flag.Bool("grok", false, "Include Grok model")
	gptFlag      = flag.Bool("gpt", false, "Include GPT model")
	claudeFlag   = flag.Bool("claude", false, "Include Claude model")
	geminiFlag   = flag.Bool("gemini", false, "Include Gemini model")
	noGrokFlag   = flag.Bool("no-grok", false, "Exclude Grok model")
	noGptFlag    = flag.Bool("no-gpt", false, "Exclude GPT model")
	noClaudeFlag = flag.Bool("no-claude", false, "Exclude Claude model")
	noGeminiFlag = flag.Bool("no-gemini", false, "Exclude Gemini model")
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}
	clients = make(map[*websocket.Conn]bool)
	clientsMutex sync.Mutex
)

func init() {
}

func main() {
	// Load keys
	loadKeys()

	ctx := context.Background()

	// Load rates
	rates := utils.LoadRates(ctx)

	// Init clients
	models.InitClients(rates)

	// Filter models - include all by default for web version
	activeModels := []*types.ModelInfo{}
	for _, mi := range models.ModelMap {
		activeModels = append(activeModels, mi)
	}

	// Check keys for active models
	for _, mi := range activeModels {
		if mi.APIKey == "" {
			log.Printf("Warning: API key for %s missing", mi.Name)
		}
	}

	// Setup Gin router
	r := gin.Default()

	// Serve static files
	r.Static("/static", "./static")

	// Routes
	r.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})

	r.GET("/ws", handleWebSocket)
	r.POST("/question", func(c *gin.Context) { handleQuestion(c, ctx, activeModels) })

	// Start server
	log.Println("Starting FAT daemon on localhost:4444")
	log.Fatal(r.Run(":4444"))
}

func broadcastMessage(message map[string]interface{}) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()

	messageBytes, _ := json.Marshal(message)

	for client := range clients {
		if err := client.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
			log.Printf("WebSocket error: %v", err)
			client.Close()
			delete(clients, client)
		}
	}
}

func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
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

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func handleQuestion(c *gin.Context, ctx context.Context, activeModels []*types.ModelInfo) {
	var req struct {
		Question string `json:"question"`
		Rounds   int    `json:"rounds"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	if req.Question == "" {
		c.JSON(400, gin.H{"error": "Question is required"})
		return
	}

	if req.Rounds < 1 || req.Rounds > 10 {
		req.Rounds = 3 // Default to 3 rounds
	}

	questionTS := time.Now().Unix()

	// Send loading messages
	for _, mi := range activeModels {
		broadcastMessage(map[string]interface{}{
			"type":   "loading",
			"model":  mi.ID,
		})
	}

	// Process question in background
	go func() {
		processQuestion(ctx, req.Question, req.Rounds, activeModels, questionTS)
	}()

	c.JSON(200, gin.H{"status": "processing"})
}

func processQuestion(ctx context.Context, question string, numRounds int, activeModels []*types.ModelInfo, questionTS int64) {
	// Clear previous responses and send round start
	broadcastMessage(map[string]interface{}{
		"type": "clear",
	})

	history := make(types.History)

	for round := 0; round < numRounds; round++ {
		broadcastMessage(map[string]interface{}{
			"type":   "round_start",
			"round":  round + 1,
			"total":  numRounds,
		})

		results := parallelCall(ctx, question, history, activeModels, round, numRounds, questionTS)

		// Wait for all models to complete this round before moving to next round
		for range activeModels {
			<-results
		}
	}

	// Ranking phase
	broadcastMessage(map[string]interface{}{
		"type": "ranking_start",
	})

	winner := rankModels(ctx, question, history, activeModels, questionTS)

	broadcastMessage(map[string]interface{}{
		"type":   "winner",
		"model":  winner,
		"answer": history[winner][len(history[winner])-1].Refined,
	})
}

func parallelCall(ctx context.Context, question string, history types.History, activeModels []*types.ModelInfo, round int, numRounds int, questionTS int64) <-chan struct{} {
	done := make(chan struct{}, len(activeModels))

	for _, mi := range activeModels {
		go func(mi *types.ModelInfo) {
			defer func() { done <- struct{}{} }()

			var prompt string
			if round == 0 {
				prompt = prompts.InitialPrompt(question)
			} else {
				context := utils.BuildContext(question, history, mi.ID)
				suggs := utils.GetSuggestionsForModel(history, mi.ID)
				if round == numRounds-1 {
					prompt = prompts.FinalPrompt(question, context, suggs, mi, round, numRounds, activeModels)
				} else {
					prompt = prompts.RefinePrompt(question, context, suggs, mi, round, numRounds, activeModels)
				}
			}
			resp, full, _, _, err := models.CallModel(ctx, mi, prompt, nil)
			if err == nil {
				utils.Log(questionTS, fmt.Sprintf("R%d", round+1), mi.Name, prompt, full)
			}

			// Send response immediately
			if err != nil {
				broadcastMessage(map[string]interface{}{
					"type":  "error",
					"model": mi.ID,
					"round": round + 1,
					"error": err.Error(),
				})
			} else {
				broadcastMessage(map[string]interface{}{
					"type":     "response",
					"model":    mi.ID,
					"round":    round + 1,
					"response": resp.Refined,
				})
				// Update history immediately for next round
				history[mi.ID] = append(history[mi.ID], resp)
			}
		}(mi)
	}

	return done
}

func rankModels(ctx context.Context, question string, history types.History, activeModels []*types.ModelInfo, questionTS int64) string {
	// Build final answers context
	finalAnswers := "Final answers from all models:\n"
	nameToId := make(map[string]string)
	for _, mi := range activeModels {
		if responses, ok := history[mi.ID]; ok && len(responses) > 0 {
			final := responses[len(responses)-1].Refined
			finalAnswers += fmt.Sprintf("Model %s:\n%s\n\n", mi.Name, final)
		}
		nameToId[mi.Name] = mi.ID
	}

	// Collect rankings from all models
	positions := make(map[string][]int)
	var wg sync.WaitGroup
	ch := make(chan struct {
		id      string
		ranking []string
		err     error
	}, len(activeModels))

	for _, mi := range activeModels {
		wg.Add(1)
		go func(mi *types.ModelInfo) {
			defer wg.Done()
			prompt := prompts.RankPrompt(question, finalAnswers, activeModels, mi)
			resp, full, _, _, err := models.CallModel(ctx, mi, prompt, nil)
			if err != nil {
				ch <- struct {
					id      string
					ranking []string
					err     error
				}{mi.ID, nil, err}
				return
			}
			utils.Log(questionTS, "rank", mi.Name, prompt, full)
			rankingStr := strings.TrimSpace(resp.Refined)
			ranking := strings.Split(rankingStr, " > ")
			for i, name := range ranking {
				ranking[i] = strings.TrimSpace(name)
			}
			ch <- struct {
				id      string
				ranking []string
				err     error
			}{mi.ID, ranking, nil}
		}(mi)
	}

	wg.Wait()
	close(ch)

	for res := range ch {
		if res.err != nil {
			continue
		}
		if len(res.ranking) == len(activeModels) {
			for i, name := range res.ranking {
				positions[name] = append(positions[name], i+1) // 1-based position
			}
		}
	}

	// Calculate average positions
	type rankInfo struct {
		name  string
		avg   float64
		first int
	}
	var ranks []rankInfo
	for name, pos := range positions {
		if len(pos) == 0 {
			continue
		}
		sum := 0
		first := 0
		for _, p := range pos {
			sum += p
			if p == 1 {
				first++
			}
		}
		avg := float64(sum) / float64(len(pos))
		ranks = append(ranks, rankInfo{name: name, avg: avg, first: first})
	}

	// Find winner: lowest avg, then most first places
	best := rankInfo{avg: 1000}
	for _, r := range ranks {
		if r.avg < best.avg || (r.avg == best.avg && r.first > best.first) {
			best = r
		}
	}

	winnerName := best.name
	winnerID, ok := nameToId[winnerName]
	if !ok {
		winnerID = activeModels[0].ID
	}

	return winnerID
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
