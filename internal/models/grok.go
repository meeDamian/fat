package models

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/meedamian/fat/internal/types"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// GrokModel implements the Model interface for Grok
type GrokModel struct {
	info *types.ModelInfo
}

// NewGrokModel creates a new Grok model instance
func NewGrokModel(info *types.ModelInfo) *GrokModel {
	return &GrokModel{info: info}
}

// Prompt implements the Model interface
func (m *GrokModel) Prompt(ctx context.Context, question string, replies map[string]string, discussion map[string][]string) (types.ModelResult, error) {
	prompt := m.formatPrompt(question, replies, discussion)

	// Build messages array
	messages := []map[string]string{{"role": "user", "content": prompt}}

	// Call Grok API
	body := map[string]any{
		"model":    m.info.Name,
		"messages": messages,
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", m.info.BaseURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return types.ModelResult{}, err
	}
	req.Header.Set("Authorization", "Bearer "+m.info.APIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return types.ModelResult{}, err
	}
	defer res.Body.Close()

	var result map[string]any
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return types.ModelResult{}, err
	}

	if res.StatusCode != 200 {
		return types.ModelResult{}, fmt.Errorf("grok API error: %v", result)
	}

	content := result["choices"].([]any)[0].(map[string]any)["message"].(map[string]any)["content"].(string)
	usage := result["usage"].(map[string]any)
	tokIn := int64(usage["prompt_tokens"].(float64))
	tokOut := int64(usage["completion_tokens"].(float64))

	reply := m.parseResponse(content)

	return types.ModelResult{
		Reply:  reply,
		TokIn:  tokIn,
		TokOut: tokOut,
	}, nil
}

// formatPrompt creates a clean prompt string
func (m *GrokModel) formatPrompt(question string, replies map[string]string, discussion map[string][]string) string {
	var b strings.Builder

	b.WriteString("You are Grok, an AI assistant in a multi-agent collaboration.\n\n")
	b.WriteString("# QUESTION\n\n")
	b.WriteString(question)
	b.WriteString("\n\n")

	if len(replies) > 0 {
		b.WriteString("# REPLIES FROM OTHER AGENTS\n\n")
		for agent, reply := range replies {
			b.WriteString(fmt.Sprintf("## %s\n\n%s\n\n", agent, reply))
		}
	}

	if len(discussion) > 0 {
		b.WriteString("# DISCUSSION HISTORY\n\n")
		for fromAgent, messages := range discussion {
			for _, msg := range messages {
				b.WriteString(fmt.Sprintf("**%s:** %s\n\n", fromAgent, msg))
			}
		}
	}

	b.WriteString(`# RESPONSE FORMAT

Provide your response in this exact format:

# ANSWER

[Your complete answer here]

# RATIONALE

[Your reasoning here]

# DISCUSSION

## With [AgentName]

[Your message to that specific agent]

## With [AnotherAgentName]

[Your message to another agent]

Only include discussion sections for agents you want to address. Be concise but thorough.`)

	return b.String()
}

// parseResponse parses the markdown response using goldmark
func (m *GrokModel) parseResponse(content string) types.Reply {
	reply := types.Reply{
		Discussion: make(map[string]string),
		RawContent: content,
	}

	// Parse markdown
	md := goldmark.New()
	reader := text.NewReader([]byte(content))
	doc := md.Parser().Parse(reader)

	var currentSection string
	var sectionContent strings.Builder
	var currentAgent string

	ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch node := node.(type) {
			case *ast.Heading:
				// Save previous section
				if currentSection != "" && sectionContent.Len() > 0 {
					m.saveSection(&reply, currentSection, strings.TrimSpace(sectionContent.String()), currentAgent)
					sectionContent.Reset()
					currentAgent = ""
				}

				if node.Level == 1 { // # headers
					textContent := string(node.Text(reader.Source()))
					switch textContent {
					case "ANSWER":
						currentSection = "answer"
					case "RATIONALE":
						currentSection = "rationale"
					case "DISCUSSION":
						currentSection = "discussion"
					}
				} else if node.Level == 2 && currentSection == "discussion" { // ## headers
					textContent := string(node.Text(reader.Source()))
					if strings.HasPrefix(textContent, "With ") {
						currentAgent = strings.TrimPrefix(textContent, "With ")
					}
				}
			case *ast.Text:
				if currentSection != "" {
					sectionContent.Write(node.Text(reader.Source()))
					sectionContent.WriteString(" ")
				}
			}
		}
		return ast.WalkContinue, nil
	})

	// Save final section
	if currentSection != "" && sectionContent.Len() > 0 {
		m.saveSection(&reply, currentSection, strings.TrimSpace(sectionContent.String()), currentAgent)
	}

	return reply
}

// saveSection saves content to the appropriate reply field
func (m *GrokModel) saveSection(reply *types.Reply, section, content, agent string) {
	content = strings.TrimSpace(content)
	if content == "" {
		return
	}

	switch section {
	case "answer":
		reply.Answer = content
	case "rationale":
		reply.Rationale = content
	case "discussion":
		if agent != "" {
			reply.Discussion[agent] = content
		}
	}
}
