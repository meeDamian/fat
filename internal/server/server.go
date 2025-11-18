package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/meedamian/fat/internal/apikeys"
	"github.com/meedamian/fat/internal/config"
	"github.com/meedamian/fat/internal/constants"
	"github.com/meedamian/fat/internal/db"
	"github.com/meedamian/fat/internal/htmlexport"
	"github.com/meedamian/fat/internal/models"
	"github.com/meedamian/fat/internal/orchestrator"
	"github.com/meedamian/fat/internal/types"
)

var (
	upgrader = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins for development
		},
	}
)

// Server manages HTTP and WebSocket connections
type Server struct {
	logger       *slog.Logger
	config       config.Config
	database     *db.DB
	orchestrator *orchestrator.Orchestrator
	clients      map[*websocket.Conn]bool
	clientsMutex sync.Mutex
}

// New creates a new Server instance
func New(logger *slog.Logger, cfg config.Config, database *db.DB) *Server {
	s := &Server{
		logger:   logger,
		config:   cfg,
		database: database,
		clients:  make(map[*websocket.Conn]bool),
	}

	// Create HTML exporter
	exporter := htmlexport.New(logger, "answers", "static")

	s.orchestrator = orchestrator.New(logger, database, s, exporter)
	return s
}

// Broadcast sends a message to all connected WebSocket clients
func (s *Server) Broadcast(message map[string]any) {
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()

	messageBytes, _ := json.Marshal(message)

	for client := range s.clients {
		if err := client.WriteMessage(websocket.TextMessage, messageBytes); err != nil {
			s.logger.Warn("websocket write failed", slog.Any("error", err))
			client.Close()
			delete(s.clients, client)
		}
	}
}

// Run starts the HTTP server
func (s *Server) Run() error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	r.Static("/static", "./static")
	r.GET("/", func(c *gin.Context) {
		c.File("./static/index.html")
	})
	r.GET("/ws", s.handleWebSocket)

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

		modelStats, err := s.database.GetAllModelStats(ctx)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		recentRequests, err := s.database.GetRecentRequests(ctx, 10)
		if err != nil {
			c.JSON(500, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{
			"model_stats":     modelStats,
			"recent_requests": recentRequests,
		})
	})

	// Models endpoint
	r.GET("/models", func(c *gin.Context) {
		familiesData := make(map[string]gin.H)

		for familyID, family := range models.ModelFamilies {
			variants := make([]gin.H, 0, len(family.Variants))
			for variantKey, variant := range family.Variants {
				variants = append(variants, gin.H{
					"key":      variantKey,
					"name":     variantKey,
					"rate_in":  variant.Rate.In,
					"rate_out": variant.Rate.Out,
				})
			}

			activeVariant := models.DefaultModels[familyID]

			familiesData[familyID] = gin.H{
				"id":       family.ID,
				"provider": family.Provider,
				"variants": variants,
				"active":   activeVariant,
			}
		}

		c.JSON(200, familiesData)
	})

	// Random question endpoint
	r.GET("/question/random", func(c *gin.Context) {
		if len(constants.SampleQuestions) == 0 {
			c.JSON(200, gin.H{"question": ""})
			return
		}
		randomIndex := rand.Intn(len(constants.SampleQuestions))
		c.JSON(200, gin.H{"question": constants.SampleQuestions[randomIndex]})
	})

	s.logger.Info("starting server", slog.String("addr", s.config.ServerAddress))
	return r.Run(s.config.ServerAddress)
}

func (s *Server) handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.logger.Error("websocket upgrade failed", slog.Any("error", err))
		return
	}

	s.clientsMutex.Lock()
	s.clients[conn] = true
	s.clientsMutex.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	defer func() {
		s.clientsMutex.Lock()
		delete(s.clients, conn)
		s.clientsMutex.Unlock()
		conn.Close()
	}()

	for {
		var msg map[string]any
		err := conn.ReadJSON(&msg)
		if err != nil {
			s.logger.Debug("websocket read error", slog.Any("error", err))
			break
		}

		msgType, ok := msg["type"].(string)
		if !ok {
			continue
		}

		switch msgType {
		case "question":
			s.handleQuestionWS(conn, ctx, msg)
		}
	}
}

func (s *Server) handleQuestionWS(conn *websocket.Conn, ctx context.Context, msg map[string]any) {
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
		rounds = 3
	}

	// Build activeModels from selected models
	selectedModels, _ := msg["models"].(map[string]any)
	activeModels := []*types.ModelInfo{}

	for familyID, family := range models.ModelFamilies {
		var variantKey string

		if selectedModels != nil {
			if selected, ok := selectedModels[familyID].(string); ok && selected != "" {
				variantKey = selected
			}
		}
		if variantKey == "" {
			variantKey = models.DefaultModels[familyID]
		}

		variant, ok := family.Variants[variantKey]
		if !ok {
			s.logger.Warn("unknown variant for family",
				slog.String("family", familyID),
				slog.String("variant", variantKey))
			continue
		}

		mi := &types.ModelInfo{
			ID:             family.ID,
			Name:           variantKey,
			MaxTok:         variant.MaxTok,
			BaseURL:        family.BaseURL,
			Logger:         s.logger.With("model", variantKey),
			RequestTimeout: s.config.ModelRequestTimeout,
		}

		if apiKey := apikeys.GetForFamily(familyID); apiKey != "" {
			mi.APIKey = apiKey
		} else {
			s.logger.Warn("api key missing for model",
				slog.String("family", familyID),
				slog.String("model", variantKey))
		}

		activeModels = append(activeModels, mi)
	}

	questionTS := time.Now().Unix()

	// Send loading messages
	for _, mi := range activeModels {
		s.Broadcast(map[string]any{
			"type":  "loading",
			"model": mi.ID,
		})
	}

	// Process question in background
	go func() {
		s.orchestrator.ProcessQuestion(ctx, question, rounds, activeModels, questionTS)
	}()
}
