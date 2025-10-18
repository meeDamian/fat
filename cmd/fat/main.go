package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/joho/godotenv"
	"github.com/meedamian/fat/internal/models"
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
)

func main() {
	// Load keys
	loadKeys()

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

	// Start server
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

	ctx := context.Background()

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

	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
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
	// Clear previous responses and send round start
	broadcastMessage(map[string]interface{}{
		"type": "clear",
	})

	// Initialize conversation state
	replies := make(map[string]string)      // agent -> latest reply
	discussion := make(map[string][]string) // fromAgent -> list of messages

	for round := 0; round < numRounds; round++ {
		broadcastMessage(map[string]interface{}{
			"type":  "round_start",
			"round": round + 1,
			"total": numRounds,
		})

		results := parallelCall(ctx, question, replies, discussion, activeModels, round, numRounds, questionTS)

		// Wait for all models to complete this round
		for range activeModels {
			result := <-results
			if result.err != nil {
				broadcastMessage(map[string]interface{}{
					"type":  "error",
					"model": result.modelID,
					"round": round + 1,
					"error": result.err.Error(),
				})
			} else {
				// Update conversation state
				replies[result.modelID] = result.reply.Answer
				for range result.reply.Discussion {
					discussion[result.modelID] = append(discussion[result.modelID], result.reply.Discussion[result.modelID])
				}

				broadcastMessage(map[string]interface{}{
					"type":     "response",
					"model":    result.modelID,
					"round":    round + 1,
					"response": result.reply.Answer,
				})
			}
		}
	}

	// Ranking phase
	broadcastMessage(map[string]interface{}{
		"type": "ranking_start",
	})

	winner := rankModels(ctx, question, replies, activeModels, questionTS)

	broadcastMessage(map[string]interface{}{
		"type":   "winner",
		"model":  winner,
		"answer": replies[winner],
	})
}

type callResult struct {
	modelID string
	reply   types.Reply
	err     error
}

func parallelCall(ctx context.Context, question string, replies map[string]string, discussion map[string][]string, activeModels []*types.ModelInfo, round int, numRounds int, questionTS int64) <-chan callResult {
	results := make(chan callResult, len(activeModels))

	for _, mi := range activeModels {
		go func(mi *types.ModelInfo) {
			defer func() {
				if r := recover(); r != nil {
					results <- callResult{modelID: mi.ID, err: fmt.Errorf("panic: %v", r)}
				}
			}()

			model := models.NewModel(mi)
			result, err := model.Prompt(ctx, question, replies, discussion)

			if err != nil {
				results <- callResult{modelID: mi.ID, err: err}
				return
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

func rankModels(ctx context.Context, question string, replies map[string]string, activeModels []*types.ModelInfo, questionTS int64) string {
	// Build final answers context
	finalAnswers := "Final answers from all models:\n"
	nameToId := make(map[string]string)
	for _, mi := range activeModels {
		if answer, ok := replies[mi.ID]; ok {
			finalAnswers += fmt.Sprintf("Model %s:\n%s\n\n", mi.Name, answer)
		}
		nameToId[mi.Name] = mi.ID
	}

	// Use a simple ranking model (could be improved)
	// For now, just pick the first model with a response
	for _, mi := range activeModels {
		if _, ok := replies[mi.ID]; ok {
			return mi.ID
		}
	}

	// Fallback to first model
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
