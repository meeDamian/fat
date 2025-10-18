package models

import (
	"context"
	"fmt"
	"strings"

	"github.com/meedamian/fat/internal/types"
	"google.golang.org/genai"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// GeminiModel implements the Model interface for Google Gemini
type GeminiModel struct {
	info   *types.ModelInfo
	client *genai.Client
}

// NewGeminiModel creates a new Gemini model instance
func NewGeminiModel(info *types.ModelInfo) *GeminiModel {
	client, _ := genai.NewClient(context.Background(), &genai.ClientConfig{APIKey: info.APIKey})
	return &GeminiModel{
		info:   info,
		client: client,
	}
}

// Prompt implements the Model interface
func (m *GeminiModel) Prompt(ctx context.Context, question string, replies map[string]string, discussion map[string][]string) (types.ModelResult, error) {
	prompt := m.formatPrompt(question, replies, discussion)

	result, err := m.client.Models.GenerateContent(ctx, m.info.Name, genai.Text(prompt), nil)
	if err != nil {
		return types.ModelResult{}, err
	}

	content := result.Text()
	reply := m.parseResponse(content)

	// Gemini doesn't provide token usage, so we estimate or set to 0
	return types.ModelResult{
		Reply:  reply,
		TokIn:  0, // Not available from Gemini API
		TokOut: 0, // Not available from Gemini API
	}, nil
}

// formatPrompt creates a clean prompt string
func (m *GeminiModel) formatPrompt(question string, replies map[string]string, discussion map[string][]string) string {
	var b strings.Builder

	b.WriteString("You are Gemini, an AI assistant in a multi-agent collaboration.\n\n")
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
func (m *GeminiModel) parseResponse(content string) types.Reply {
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
func (m *GeminiModel) saveSection(reply *types.Reply, section, content, agent string) {
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
