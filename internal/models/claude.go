package models

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	an "github.com/anthropics/anthropic-sdk-go/option"
	"github.com/meedamian/fat/internal/types"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// ClaudeModel implements the Model interface for Anthropic Claude
type ClaudeModel struct {
	info   *types.ModelInfo
	client anthropic.Client
}

// NewClaudeModel creates a new Claude model instance
func NewClaudeModel(info *types.ModelInfo) *ClaudeModel {
	client := anthropic.NewClient(an.WithAPIKey(info.APIKey), an.WithMaxRetries(3))
	return &ClaudeModel{
		info:   info,
		client: client,
	}
}

// Prompt implements the Model interface
func (m *ClaudeModel) Prompt(ctx context.Context, question string, replies map[string]string, discussion map[string][]string) (types.ModelResult, error) {
	prompt := m.formatPrompt(question, replies, discussion)

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model("claude-3-5-haiku-latest"),
		MaxTokens: 1024,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	}

	result, err := m.client.Messages.New(ctx, params)
	if err != nil {
		return types.ModelResult{}, err
	}

	content := result.Content[0].Text
	reply := m.parseResponse(content)

	return types.ModelResult{
		Reply:  reply,
		TokIn:  result.Usage.InputTokens,
		TokOut: result.Usage.OutputTokens,
	}, nil
}

// formatPrompt creates a clean prompt string
func (m *ClaudeModel) formatPrompt(question string, replies map[string]string, discussion map[string][]string) string {
	var b strings.Builder

	b.WriteString("You are Claude, an AI assistant in a multi-agent collaboration.\n\n")
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
func (m *ClaudeModel) parseResponse(content string) types.Reply {
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
func (m *ClaudeModel) saveSection(reply *types.Reply, section, content, agent string) {
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
