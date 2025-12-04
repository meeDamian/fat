package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
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
	staticFS     fs.FS
	startTime    time.Time
}

// New creates a new Server instance
func New(logger *slog.Logger, cfg config.Config, database *db.DB, staticFS fs.FS) *Server {
	s := &Server{
		logger:    logger,
		config:    cfg,
		database:  database,
		clients:   make(map[*websocket.Conn]bool),
		staticFS:  staticFS,
		startTime: time.Now(),
	}

	// Create HTML exporter
	exporter := htmlexport.New(logger, "web/static")

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

// slogMiddleware creates a Gin middleware that logs HTTP requests using slog
func (s *Server) slogMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Process request
		c.Next()

		// Log after request is processed
		duration := time.Since(start)
		status := c.Writer.Status()

		// Choose log level based on status code
		logFunc := s.logger.Info
		if status >= 500 {
			logFunc = s.logger.Error
		} else if status >= 400 {
			logFunc = s.logger.Warn
		}

		logFunc("http request",
			slog.String("method", method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Duration("duration", duration),
			slog.String("ip", c.ClientIP()),
		)
	}
}

// Run starts the HTTP server
func (s *Server) Run() error {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(s.slogMiddleware())

	// Serve embedded static files
	staticSubFS, err := fs.Sub(s.staticFS, "static")
	if err != nil {
		return err
	}
	r.StaticFS("/static", http.FS(staticSubFS))

	// Serve index.html from embedded files
	r.GET("/", func(c *gin.Context) {
		data, err := fs.ReadFile(s.staticFS, "static/index.html")
		if err != nil {
			c.String(500, "Failed to load index.html")
			return
		}
		c.Data(200, "text/html; charset=utf-8", data)
	})

	// Serve /h/ directory with directory listing
	r.GET("/h/*filepath", func(c *gin.Context) {
		filepath := c.Param("filepath")
		if filepath == "" || filepath == "/" {
			// Generate directory listing
			s.serveDirectoryListing(c, "h")
			return
		}
		// Serve static file
		c.File("h" + filepath)
	})

	r.GET("/ws", s.handleWebSocket)

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
			"uptime": time.Since(s.startTime).String(),
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

	// Shutdown endpoints
	r.GET("/die/now", func(c *gin.Context) {
		s.logger.Warn("received die/now request, exiting immediately")
		os.Exit(1)
	})

	r.GET("/die", func(c *gin.Context) {
		if s.orchestrator.IsProcessing() {
			c.JSON(423, gin.H{"error": "processing in progress"})
			return
		}
		s.logger.Info("received die request, exiting")
		os.Exit(1)
	})

	r.GET("/perish", func(c *gin.Context) {
		s.logger.Warn("received perish request, exiting immediately")
		os.Exit(0)
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

// serveDirectoryListing generates an HTML page listing all files in the h/ directory
func (s *Server) serveDirectoryListing(c *gin.Context, baseDir string) {
	type FileEntry struct {
		Path    string
		Name    string
		ModTime time.Time
		Size    int64
	}

	type DateGroup struct {
		Date  string
		Files []FileEntry
	}

	// Read directory structure
	groups := make(map[string][]FileEntry)

	err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".html") {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		// Extract date from path (h/YYYY-MM-DD/file.html)
		parts := strings.Split(path, string(os.PathSeparator))
		date := "unknown"
		if len(parts) >= 2 {
			date = parts[1]
		}

		groups[date] = append(groups[date], FileEntry{
			Path:    "/" + path,
			Name:    filepath.Base(path),
			ModTime: info.ModTime(),
			Size:    info.Size(),
		})

		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		c.String(500, "Error reading directory: %v", err)
		return
	}

	// Sort dates descending (newest first)
	dates := make([]string, 0, len(groups))
	for date := range groups {
		dates = append(dates, date)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(dates)))

	// Sort files within each group by name (descending = newest first)
	for date := range groups {
		sort.Slice(groups[date], func(i, j int) bool {
			return groups[date][i].Name > groups[date][j].Name
		})
	}

	// Build HTML
	var html strings.Builder
	html.WriteString(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Nexus - Exported Sessions</title>
    <style>
        :root { --bg: #0a0a0f; --text: #e4e4e7; --muted: #71717a; --accent: #7c5cff; }
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body { background: var(--bg); color: var(--text); font-family: system-ui, sans-serif; padding: 40px 20px; max-width: 900px; margin: 0 auto; }
        h1 { font-size: 2em; margin-bottom: 8px; }
        .tagline { color: var(--muted); margin-bottom: 40px; }
        .date-group { margin-bottom: 32px; }
        .date-header { color: var(--accent); font-size: 1.1em; font-weight: 600; margin-bottom: 12px; padding-bottom: 8px; border-bottom: 1px solid rgba(255,255,255,0.1); }
        .file-list { list-style: none; }
        .file-list li { margin-bottom: 8px; }
        .file-list a { color: var(--text); text-decoration: none; display: block; padding: 12px 16px; background: rgba(255,255,255,0.03); border-radius: 8px; transition: all 0.2s; }
        .file-list a:hover { background: rgba(124, 92, 255, 0.15); transform: translateX(4px); }
        .file-name { font-weight: 500; }
        .file-meta { color: var(--muted); font-size: 0.85em; margin-top: 4px; }
        .empty { color: var(--muted); font-style: italic; }
    </style>
</head>
<body>
    <h1>📄 Exported Sessions</h1>
    <p class="tagline">Static HTML snapshots of Nexus conversations</p>
`)

	if len(dates) == 0 {
		html.WriteString(`    <p class="empty">No exports yet. Run some questions and they'll appear here!</p>`)
	} else {
		for _, date := range dates {
			html.WriteString(fmt.Sprintf(`    <div class="date-group">
        <div class="date-header">📅 %s</div>
        <ul class="file-list">
`, date))
			for _, f := range groups[date] {
				sizeKB := float64(f.Size) / 1024
				html.WriteString(fmt.Sprintf(`            <li><a href="%s">
                <div class="file-name">%s</div>
                <div class="file-meta">%.1f KB</div>
            </a></li>
`, f.Path, f.Name, sizeKB))
			}
			html.WriteString(`        </ul>
    </div>
`)
		}
	}

	html.WriteString(`</body>
</html>`)

	c.Data(200, "text/html; charset=utf-8", []byte(html.String()))
}
