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
	"sort"
	"strings"
	"time"

	"github.com/meedamian/fat/internal/db"
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
	Question        string
	QuestionTS      int64    // Unix timestamp for directory
	GoldIDs         []string // Models that won gold (can be multiple if tied)
	SilverIDs       []string // Models that won silver
	BronzeIDs       []string // Models that won bronze
	Replies         map[string]types.Reply
	AllRoundReplies map[string]map[int]db.ModelRound // Model ID -> Round -> ModelRound
	Models          []*types.ModelInfo
	Metrics         map[string]any
	RoundCounts     map[string]int    // Model ID -> number of rounds completed
	ModelCosts      map[string]string // Model ID -> formatted cost string
	ModelScores     map[string]int    // Model ID -> ranking score
	Discussions     []DiscussionPair
	Timestamp       string
	PageTitle       string // Formatted title for HTML <title> tag
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

	// Generate HTML
	html, err := e.renderHTML(data)
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

func (e *Exporter) renderHTML(data ExportData) (string, error) {
	// Read CSS from static directory
	cssPath := filepath.Join(e.staticDir, "style.css")
	cssBytes, err := os.ReadFile(cssPath)
	if err != nil {
		return "", fmt.Errorf("failed to read CSS file: %w", err)
	}

	// Format model names for display
	modelNames := make(map[string]string)
	for _, model := range data.Models {
		modelNames[model.ID] = formatModelName(model.ID)
	}

	// Calculate cost gradient colors
	costColors := make(map[string]string)
	if len(data.ModelCosts) > 0 {
		costValues := make(map[string]float64)
		var minCost, maxCost float64
		first := true
		for modelID, costStr := range data.ModelCosts {
			var cost float64
			fmt.Sscanf(costStr, "$%f", &cost)
			costValues[modelID] = cost
			if first {
				minCost = cost
				maxCost = cost
				first = false
			} else {
				if cost < minCost {
					minCost = cost
				}
				if cost > maxCost {
					maxCost = cost
				}
			}
		}

		costRange := maxCost - minCost
		for modelID, cost := range costValues {
			if cost == 0 {
				continue
			}

			var position float64
			if costRange > 0 {
				position = (cost - minCost) / costRange
			}

			var r, g, b int
			if position < 0.5 {
				t := position * 2
				r = int(129 + (255-129)*t)
				g = int(199 + (235-199)*t)
				b = int(132 * (1 - t))
			} else {
				t := (position - 0.5) * 2
				r = 255
				g = int(235 * (1 - t))
				b = 0
			}

			costColors[modelID] = fmt.Sprintf("background-color: rgba(%d, %d, %d, 0.2); color: rgb(%d, %d, %d);", r, g, b, r, g, b)
		}
	}

	// Sort models by score
	sortedModels := make([]*types.ModelInfo, len(data.Models))
	copy(sortedModels, data.Models)
	sort.Slice(sortedModels, func(i, j int) bool {
		scoreI := data.ModelScores[sortedModels[i].ID]
		scoreJ := data.ModelScores[sortedModels[j].ID]
		return scoreI > scoreJ
	})

	// Prepare complete data structure for JavaScript
	exportData := map[string]any{
		"question":        data.Question,
		"pageTitle":       data.PageTitle,
		"goldIDs":         data.GoldIDs,
		"silverIDs":       data.SilverIDs,
		"bronzeIDs":       data.BronzeIDs,
		"replies":         data.Replies,
		"allRoundReplies": data.AllRoundReplies,
		"models":          sortedModels,
		"modelNames":      modelNames,
		"metrics":         data.Metrics,
		"roundCounts":     data.RoundCounts,
		"modelCosts":      data.ModelCosts,
		"costColors":      costColors,
		"modelScores":     data.ModelScores,
		"discussions":     data.Discussions,
		"timestamp":       data.Timestamp,
	}

	dataJSON, err := json.Marshal(exportData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	// Create minimal template with embedded CSS and data
	tmpl, err := template.New("export").Parse(htmlTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]any{
		"CSS":  template.CSS(cssBytes),
		"DATA": template.JS(dataJSON),
	}); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}

	return buf.String(), nil
}

func formatModelName(id string) string {
	switch id {
	case "grok":
		return "Grok"
	case "gpt":
		return "GPT"
	case "gemini":
		return "Gemini"
	case "claude":
		return "Claude"
	case "deepseek":
		return "DeepSeek"
	case "mistral":
		return "Mistral"
	default:
		return id
	}
}

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title id="pageTitle">Loading...</title>
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
    background: rgba(15, 23, 42, 0.5);
    border: 1px solid var(--border-subtle);
    border-radius: 24px;
    padding: clamp(24px, 4vw, 36px);
    margin-bottom: 32px;
    box-shadow: inset 0 2px 4px rgba(0, 0, 0, 0.2);
}

.static-question h2 {
    font-size: 13px;
    font-weight: 700;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.08em;
    margin: 0 0 16px 0;
}

.static-question p {
    font-size: 20px;
    line-height: 1.6;
    color: var(--text-main);
    margin: 0;
    font-weight: 500;
    white-space: pre-wrap;
}

.model-selector {
    display: none !important;
}

.model-chip {
    font-size: 11px;
    padding: 4px 8px;
    border-radius: 6px;
    background: rgba(255, 255, 255, 0.05);
    color: var(--text-muted);
    font-weight: 600;
    white-space: nowrap;
    text-transform: uppercase;
    letter-spacing: 0.05em;
}

.round-dot.filled {
    background: var(--accent-primary) !important;
    box-shadow: 0 0 8px rgba(56, 189, 248, 0.4);
}

.model-card-header {
    display: flex !important;
    flex-wrap: wrap !important;
    align-items: center !important;
    justify-content: space-between !important;
    gap: 12px !important;
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

/* Centered medal */
.model-medal-center {
    display: flex;
    justify-content: center;
    align-items: center;
    padding: 12px 0 8px 0;
}

.model-medal {
    font-size: 48px;
    line-height: 1;
    filter: drop-shadow(0 4px 8px rgba(0,0,0,0.3));
}

/* Round dots are now interactive in static export */
.round-dot.filled {
    cursor: pointer !important;
    transition: all 0.2s ease;
}

.round-dot.filled:hover {
    transform: scale(1.2);
    background: #fff !important;
}

/* Ensure cost is visible with proper styling */
.model-cost {
    display: inline-block !important;
    visibility: visible !important;
    opacity: 1 !important;
    font-size: 12px;
    padding: 3px 8px;
    border-radius: 999px;
    font-weight: 500;
    font-family: 'SF Mono', 'Monaco', 'Consolas', monospace;
}

/* Discussion section spacing fixes */
.discussion-pair {
    margin-bottom: 32px;
}

.discussion-pair:last-child {
    margin-bottom: 0;
}

.discussion-pair-header {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    font-size: 13px;
    font-weight: 600;
    color: rgba(148, 163, 184, 0.9);
    margin-bottom: 16px;
    text-align: center;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    line-height: 1.5;
}

.discussion-filters {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-bottom: 24px;
}

.discussion-filter-chip {
    padding: 8px 16px;
    background: rgba(15, 13, 30, 0.6);
    border: 1px solid rgba(124, 92, 255, 0.3);
    border-radius: 20px;
    color: rgba(237, 236, 255, 0.85);
    font-size: 13px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s ease;
    font-family: inherit;
}

.discussion-filter-chip:hover {
    background: rgba(124, 92, 255, 0.15);
    border-color: rgba(124, 92, 255, 0.5);
    color: rgba(255, 255, 255, 0.95);
}

.discussion-filter-chip.active {
    background: rgba(124, 92, 255, 0.25);
    border-color: rgba(124, 92, 255, 0.7);
    color: rgba(255, 255, 255, 1);
    font-weight: 600;
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
    
    <script>
    // All data for rendering
    const DATA = {{.DATA}};
    </script>
</head>
<body>
    <div class="app-shell">
        <header class="hero compact">
            <h1>Nexus</h1>
            <p class="tagline">Collaborative Intelligence.</p>
        </header>

        <main class="workspace">
            <section class="control-panel" aria-label="Question">
                <div class="static-question">
                    <h2>Question</h2>
                    <p id="questionText"></p>
                </div>
            </section>

            <section id="conversationBoard" class="board">
                <div class="models-layout">
                    <div id="heroStage" class="hero-stage"></div>
                    <div id="galleryStage" class="gallery-stage">
                        <!-- Model cards will be rendered by JavaScript -->
                    </div>
                </div>
            </section>

            <section id="discussionsSection" class="discussions-section" style="display: none;">
                <h2>Agent Discussions</h2>
                <div id="discussionFilters" class="discussion-filters">
                    <!-- Filter chips will be rendered by JavaScript -->
                </div>
                <div id="discussionsContainer" class="discussions-container">
                    <!-- Discussions will be rendered by JavaScript -->
                </div>
            </section>
        </main>

        <footer class="footer">
            <span class="footer-text">Made with 🥩 and ☕️ by <a href="https://x.com/meeDamian"><strong>meeDamian</strong></a>. Generated <span id="timestamp"></span></span>
        </footer>
    </div>
    
    <script>
    // Render page on load
    document.addEventListener('DOMContentLoaded', function() {
        // Set page title
        document.getElementById('pageTitle').textContent = DATA.pageTitle + ' - Nexus';
        document.title = DATA.pageTitle + ' - Nexus';
        
        // Set question
        document.getElementById('questionText').textContent = DATA.question;
        
        // Set timestamp
        document.getElementById('timestamp').textContent = DATA.timestamp;
        
        // Render model cards
        const galleryStage = document.getElementById('galleryStage');
        DATA.models.forEach(model => {
            const reply = DATA.replies[model.ID];
            const isGold = DATA.goldIDs.includes(model.ID);
            const isSilver = DATA.silverIDs.includes(model.ID);
            const isBronze = DATA.bronzeIDs.includes(model.ID);
            
            let classes = 'model-card';
            if (isGold) classes += ' winner';
            else if (isSilver) classes += ' runner-up';
            else if (isBronze) classes += ' bronze';
            
            const card = document.createElement('article');
            card.className = classes;
            card.id = model.ID;
            card.dataset.model = model.ID;
            
            // Medal
            let medalHTML = '';
            if (isGold || isSilver || isBronze) {
                const medal = isGold ? '🏆' : isSilver ? '🥈' : '🥉';
                medalHTML = '<div class="model-medal-center"><span class="model-medal">' + medal + '</span></div>';
            }
            
            // Cost
            let costHTML = '';
            if (DATA.modelCosts[model.ID]) {
                const costStyle = DATA.costColors[model.ID] || '';
                costHTML = '<span class="model-cost" style="' + costStyle + '">' + DATA.modelCosts[model.ID] + '</span>';
            }
            
            // Round dots
            const roundCount = DATA.roundCounts[model.ID] || 0;
            let dotsHTML = '';
            for (let i = 0; i < roundCount; i++) {
                dotsHTML += '<span class="round-dot filled"></span>';
            }
            
            // Answer/rationale
            let outputHTML = '';
            if (reply) {
                outputHTML = '<p class="answer-text">' + escapeHTML(reply.Answer) + '</p>';
                if (reply.Rationale) {
                    outputHTML += '<p class="rationale-text">' + escapeHTML(reply.Rationale) + '</p>';
                }
            } else {
                outputHTML = '<p class="placeholder">No response</p>';
            }
            
            card.innerHTML = 
                medalHTML +
                '<header class="model-card-header">' +
                    '<div class="model-header-left">' +
                        '<span class="model-name">' + DATA.modelNames[model.ID] + '</span>' +
                        '<span class="model-chip">' + escapeHTML(model.Name) + '</span>' +
                    '</div>' +
                    '<div class="model-header-right">' +
                        costHTML +
                        '<span class="model-provider">' + provider + '</span>' +
                    '</div>' +
                '</header>' +
                '<div class="round-progress" data-model="' + model.ID + '">' +
                    dotsHTML +
                '</div>' +
                '<div class="model-output">' +
                    outputHTML +
                '</div>';
            
            galleryStage.appendChild(card);
        });
        
        // Render discussions with filtering
        let activeDiscussionFilter = null;
        
        function renderDiscussions() {
            const discussionsContainer = document.getElementById('discussionsContainer');
            discussionsContainer.innerHTML = '';
            
            // Filter discussions based on active filter
            const filteredDiscussions = activeDiscussionFilter
                ? DATA.discussions.filter(pair => pair.Header.includes(activeDiscussionFilter))
                : DATA.discussions;
            
            filteredDiscussions.forEach(pair => {
                const pairDiv = document.createElement('div');
                pairDiv.className = 'discussion-pair';
                
                const headerDiv = document.createElement('div');
                headerDiv.className = 'discussion-pair-header';
                headerDiv.textContent = pair.Header;
                pairDiv.appendChild(headerDiv);
                
                const conversationDiv = document.createElement('div');
                conversationDiv.className = 'discussion-conversation';
                
                pair.Messages.forEach(msg => {
                    const msgDiv = document.createElement('div');
                    msgDiv.className = 'discussion-message';
                    
                    const metaSpan = document.createElement('span');
                    metaSpan.className = 'discussion-meta';
                    metaSpan.textContent = msg.Meta;
                    msgDiv.appendChild(metaSpan);
                    
                    const textDiv = document.createElement('div');
                    textDiv.className = 'discussion-text';
                    textDiv.textContent = msg.Text;
                    msgDiv.appendChild(textDiv);
                    
                    conversationDiv.appendChild(msgDiv);
                });
                
                pairDiv.appendChild(conversationDiv);
                discussionsContainer.appendChild(pairDiv);
            });
        }
        
        if (DATA.discussions && DATA.discussions.length > 0) {
            const discussionsSection = document.getElementById('discussionsSection');
            const discussionFilters = document.getElementById('discussionFilters');
            discussionsSection.style.display = '';
            
            // Build list of unique models in discussions
            const modelsInDiscussions = new Set();
            DATA.discussions.forEach(pair => {
                // Extract model names from header (e.g., "Grok ↔ GPT")
                const models = pair.Header.split(' ↔ ');
                models.forEach(name => modelsInDiscussions.add(name.trim()));
            });
            
            // Add "All" filter chip
            const allChip = document.createElement('button');
            allChip.className = 'discussion-filter-chip active';
            allChip.textContent = 'All';
            allChip.addEventListener('click', () => {
                activeDiscussionFilter = null;
                document.querySelectorAll('.discussion-filter-chip').forEach(c => c.classList.remove('active'));
                allChip.classList.add('active');
                renderDiscussions();
            });
            discussionFilters.appendChild(allChip);
            
            // Add filter chip for each model
            Array.from(modelsInDiscussions).sort().forEach(modelName => {
                const chip = document.createElement('button');
                chip.className = 'discussion-filter-chip';
                chip.textContent = modelName;
                chip.addEventListener('click', () => {
                    activeDiscussionFilter = modelName;
                    document.querySelectorAll('.discussion-filter-chip').forEach(c => c.classList.remove('active'));
                    chip.classList.add('active');
                    renderDiscussions();
                });
                discussionFilters.appendChild(chip);
            });
            
            // Initial render
            renderDiscussions();
        }
        
        // Add round dot interactivity
        const allRoundReplies = DATA.allRoundReplies;
        const currentRounds = {};
        
        // Initialize all models to their final round
        document.querySelectorAll('.model-card').forEach(card => {
            const modelId = card.dataset.model;
            const roundDots = card.querySelectorAll('.round-dot.filled');
            currentRounds[modelId] = roundDots.length;
        });
        
        // Add click handlers to round dots
        document.querySelectorAll('.round-progress').forEach(progressBar => {
            const modelId = progressBar.dataset.model;
            const dots = progressBar.querySelectorAll('.round-dot.filled');
            
            dots.forEach((dot, index) => {
                const roundNumber = index + 1;
                dot.style.cursor = 'pointer';
                dot.title = 'Click to view round ' + roundNumber;
                
                dot.addEventListener('click', () => {
                    if (!allRoundReplies[modelId] || !allRoundReplies[modelId][roundNumber]) {
                        return;
                    }
                    
                    const roundReply = allRoundReplies[modelId][roundNumber];
                    const card = progressBar.closest('.model-card');
                    const answerText = card.querySelector('.answer-text');
                    const rationaleText = card.querySelector('.rationale-text');
                    
                    if (answerText) {
                        answerText.textContent = roundReply.Answer;
                    }
                    if (rationaleText) {
                        rationaleText.textContent = roundReply.Rationale;
                    } else if (roundReply.Rationale) {
                        const modelOutput = card.querySelector('.model-output');
                        const rationaleP = document.createElement('p');
                        rationaleP.className = 'rationale-text';
                        rationaleP.textContent = roundReply.Rationale;
                        modelOutput.appendChild(rationaleP);
                    }
                    
                    // Highlight the selected dot
                    dots.forEach((d, i) => {
                        if (i === index) {
                            d.style.background = 'rgba(255, 215, 0, 1)';
                            d.style.boxShadow = '0 0 8px rgba(255, 215, 0, 0.6)';
                        } else {
                            d.style.background = '';
                            d.style.boxShadow = '';
                        }
                    });
                    
                    currentRounds[modelId] = roundNumber;
                });
            });
        });
    });
    
    // Helper function to escape HTML
    function escapeHTML(str) {
        if (!str) return '';
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }
    </script>
</body>
</html>
`
