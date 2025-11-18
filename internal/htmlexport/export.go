package htmlexport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/meedamian/fat/internal/types"
)

type Exporter struct {
	logger     *slog.Logger
	answersDir string
	staticDir  string
}

func New(logger *slog.Logger, answersDir, staticDir string) *Exporter {
	return &Exporter{
		logger:     logger,
		answersDir: answersDir,
		staticDir:  staticDir,
	}
}

type ExportData struct {
	Question    string
	QuestionTS  int64    // Unix timestamp for directory
	GoldIDs     []string // Models that won gold (can be multiple if tied)
	SilverIDs   []string // Models that won silver
	BronzeIDs   []string // Models that won bronze
	Replies     map[string]types.Reply
	Models      []*types.ModelInfo
	Metrics     map[string]any
	RoundCounts map[string]int    // Model ID -> number of rounds completed
	ModelCosts  map[string]string // Model ID -> formatted cost string
	Discussions []DiscussionPair
	Timestamp   string
	PageTitle   string // Formatted title for HTML <title> tag
}

type DiscussionPair struct {
	Header   string
	Messages []DiscussionMessage
}

type DiscussionMessage struct {
	Meta string
	Text string
}

// GenerateFilename creates a filename and page title from the question
// Returns filename (without .html extension) and page title
func (e *Exporter) GenerateFilename(ctx context.Context, question string) (string, string, error) {
	// Use fallback approach (simple and reliable)
	filename := e.fallbackFilename(question)
	// Create a title by capitalizing first letter of each word
	words := strings.Fields(question)
	if len(words) > 8 {
		words = words[:8]
	}
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	title := strings.Join(words, " ")
	if len(title) > 60 {
		title = title[:60] + "..."
	}
	return filename, title, nil
}

func (e *Exporter) fallbackFilename(question string) string {
	// Simple fallback: take first few words
	words := strings.Fields(question)
	if len(words) > 5 {
		words = words[:5]
	}
	filename := strings.ToLower(strings.Join(words, "-"))
	filename = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(filename, "-")
	filename = regexp.MustCompile(`-+`).ReplaceAllString(filename, "-")
	filename = strings.Trim(filename, "-")
	if len(filename) > 50 {
		filename = filename[:50]
	}
	if len(filename) == 0 {
		filename = fmt.Sprintf("question-%d", time.Now().Unix())
	}
	return filename
}

// Export generates and saves a static HTML file
func (e *Exporter) Export(ctx context.Context, data ExportData) error {
	// Generate filename and page title
	filenameBase, pageTitle, err := e.GenerateFilename(ctx, data.Question)
	if err != nil {
		return fmt.Errorf("generate filename: %w", err)
	}
	filename := filenameBase + ".html"

	// Set page title in data
	data.PageTitle = pageTitle

	// Read CSS from static directory
	cssPath := filepath.Join(e.staticDir, "style.css")
	cssBytes, err := os.ReadFile(cssPath)
	if err != nil {
		return fmt.Errorf("read CSS: %w", err)
	}

	// Generate HTML
	html, err := e.generateHTML(data, cssBytes)
	if err != nil {
		return fmt.Errorf("generate HTML: %w", err)
	}

	// Use existing timestamp-based directory structure
	targetDir := filepath.Join(e.answersDir, fmt.Sprintf("%d", data.QuestionTS))

	// Ensure directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("create directory: %w", err)
	}

	// Write file
	outputPath := filepath.Join(targetDir, filename)
	if err := os.WriteFile(outputPath, []byte(html), 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	e.logger.Info("static HTML exported", slog.String("path", outputPath))
	return nil
}

func (e *Exporter) generateHTML(data ExportData, cssBytes []byte) (string, error) {
	tmpl := template.Must(template.New("export").Funcs(template.FuncMap{
		"json": func(v any) string {
			b, _ := json.Marshal(v)
			return string(b)
		},
		"modelChip": func(modelID string, models []*types.ModelInfo) string {
			for _, m := range models {
				if m.ID == modelID {
					return m.Name
				}
			}
			return modelID
		},
		"until": func(count int) []int {
			result := make([]int, count)
			for i := range count {
				result[i] = i
			}
			return result
		},
		"contains": func(list []string, item string) bool {
			for _, v := range list {
				if v == item {
					return true
				}
			}
			return false
		},
		"sortModels": func(models []*types.ModelInfo) []*types.ModelInfo {
			// Desired order: grok, gpt, gemini, claude, deepseek, mistral
			order := []string{"grok", "gpt", "gemini", "claude", "deepseek", "mistral"}
			sorted := make([]*types.ModelInfo, 0, len(models))
			for _, id := range order {
				for _, m := range models {
					if m.ID == id {
						sorted = append(sorted, m)
						break
					}
				}
			}
			return sorted
		},
	}).Parse(htmlTemplate))

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]any{
		"Question":    data.Question,
		"PageTitle":   data.PageTitle,
		"GoldIDs":     data.GoldIDs,
		"SilverIDs":   data.SilverIDs,
		"BronzeIDs":   data.BronzeIDs,
		"Replies":     data.Replies,
		"Models":      data.Models,
		"Metrics":     data.Metrics,
		"RoundCounts": data.RoundCounts,
		"ModelCosts":  data.ModelCosts,
		"Discussions": data.Discussions,
		"Timestamp":   data.Timestamp,
		"CSS":         template.CSS(cssBytes),
	}); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.PageTitle}} - Sixfold</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;500;600&display=swap" rel="stylesheet">
    <style>
{{.CSS}}

/* Additional overrides for static version */
.connection-status {
    display: none !important;
}

.control-inputs {
    display: none !important;
}

.static-question {
    background: linear-gradient(135deg, rgba(15, 23, 42, 0.98), rgba(30, 41, 59, 0.95));
    border: 1px solid rgba(51, 65, 85, 0.9);
    border-radius: 18px;
    padding: clamp(24px, 5vw, 36px);
    margin-bottom: 32px;
    box-shadow: 0 12px 40px rgba(15, 23, 42, 0.8);
}

.static-question h2 {
    font-size: 14px;
    font-weight: 700;
    color: rgba(148, 163, 184, 0.9);
    text-transform: uppercase;
    letter-spacing: 0.08em;
    margin: 0 0 16px 0;
}

.static-question p {
    font-size: 20px;
    line-height: 1.6;
    color: rgba(255, 255, 255, 0.95);
    margin: 0;
    font-weight: 500;
}

.model-selector {
    display: none !important;
}

.model-chip {
    font-size: 12px;
    padding: 4px 10px;
    border-radius: 999px;
    background: rgba(124, 92, 255, 0.25);
    color: rgba(167, 139, 250, 1);
    font-weight: 500;
    white-space: nowrap;
    font-family: 'SF Mono', 'Monaco', 'Consolas', monospace;
}

.round-dot.filled {
    background: rgba(124, 92, 255, 1) !important;
}

.model-card-header {
    display: flex !important;
    align-items: center !important;
    justify-content: space-between !important;
}

.model-status {
    position: static !important;
    margin: 0 !important;
}

.model-card.bronze {
    border-color: #cd7f32 !important;
}

/* Align discussion text to match bubble side */
.discussion-message:nth-child(odd) {
    text-align: left !important;
}

.discussion-message:nth-child(even) {
    text-align: right !important;
}

/* Preserve newlines in all text content */
.answer-text,
.rationale-text,
.discussion-text {
    white-space: pre-wrap !important;
}

/* Hero layout - move winners to top in narrow view */
@media (max-width: 768px) {
    .gallery-stage {
        display: flex !important;
        flex-direction: column !important;
    }
    
    .model-card.winner {
        order: -2 !important;
    }
    
    .model-card.runner-up {
        order: -1 !important;
    }
}
    </style>
</head>
<body>
    <div class="app-shell">
        <header class="hero compact">
            <h1>Sixfold</h1>
            <p class="tagline">Six AIs debate. One answer wins.</p>
        </header>

        <main class="workspace">
            <section class="control-panel" aria-label="Question">
                <div class="static-question">
                    <h2>Question</h2>
                    <p>{{.Question}}</p>
                </div>
            </section>

            <section id="conversationBoard" class="board">
                <div class="models-layout">
                    <div id="heroStage" class="hero-stage"></div>
                    <div id="galleryStage" class="gallery-stage">
                        {{range $idx, $model := sortModels .Models}}
                        {{$reply := index $.Replies $model.ID}}
                        {{$isGold := contains $.GoldIDs $model.ID}}
                        {{$isSilver := contains $.SilverIDs $model.ID}}
                        {{$isBronze := contains $.BronzeIDs $model.ID}}
                        <article class="model-card{{if $isGold}} winner{{else if $isSilver}} runner-up{{else if $isBronze}} bronze{{end}}" id="{{$model.ID}}" data-model="{{$model.ID}}">
                            <header class="model-card-header">
                                <div class="model-header-left">
                                    <span class="model-name">{{$model.ID}}</span>
                                    <span class="model-chip">{{modelChip $model.ID $.Models}}</span>
                                </div>
                                <span class="model-status visible" aria-hidden="true">{{if $isGold}}🏆{{else if $isSilver}}🥈{{else if $isBronze}}🥉{{end}}</span>
                                <div class="model-header-right">
                                    {{$cost := index $.ModelCosts $model.ID}}
                                    {{if $cost}}<span class="model-cost" data-model="{{$model.ID}}">{{$cost}}</span>{{end}}
                                </div>
                            </header>
                            <div class="round-progress" data-model="{{$model.ID}}">
                                {{$count := index $.RoundCounts $model.ID}}
                                {{range $i := until $count}}
                                <span class="round-dot filled"></span>
                                {{end}}
                            </div>
                            <div class="model-output">
                                {{if $reply}}
                                <p class="answer-text">{{$reply.Answer}}</p>
                                {{if $reply.Rationale}}
                                <p class="rationale-text">{{$reply.Rationale}}</p>
                                {{end}}
                                {{else}}
                                <p class="placeholder">No response</p>
                                {{end}}
                            </div>
                        </article>
                        {{end}}
                    </div>
                </div>
            </section>

            {{if .Discussions}}
            <section id="discussionsSection" class="discussions-section">
                <h2>Agent Discussions</h2>
                {{range .Discussions}}
                <div class="discussion-pair">
                    <div class="discussion-pair-header">{{.Header}}</div>
                    <div class="discussion-conversation">
                        {{range .Messages}}
                        <div class="discussion-message">
                            <span class="discussion-meta">{{.Meta}}</span>
                            <div class="discussion-text">{{.Text}}</div>
                        </div>
                        {{end}}
                    </div>
                </div>
                {{end}}
            </section>
            {{end}}
        </main>

        <footer class="footer">
            <span class="footer-text">Made with 🥩 and ☕️ by <a href="https://x.com/meeDamian"><strong>meeDamian</strong></a>. Generated {{.Timestamp}}</span>
        </footer>
    </div>
</body>
</html>
`
