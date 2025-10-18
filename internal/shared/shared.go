package shared

import (
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"github.com/meedamian/fat/internal/types"
)

// FormatPrompt creates a standardized prompt for all models
func FormatPrompt(modelName, question string, meta types.Meta, replies map[string]string, discussion map[string][]string) string {
	var b strings.Builder

	// Agent list
	otherAgentsStr := "none"
	if len(meta.OtherAgents) > 0 {
		otherAgentsStr = strings.Join(meta.OtherAgents, ", ")
	}

	agentCount := len(meta.OtherAgents) + 1
	b.WriteString(fmt.Sprintf("You are agent %s in a %d-agent collaboration on the original question below. Other agents present are: %s. This is round %d of %d.\n\n", modelName, agentCount, otherAgentsStr, meta.Round, meta.TotalRounds))
	b.WriteString("--- PROMPT ---\n\n")
	b.WriteString("# QUESTION\n\n")
	b.WriteString(question)
	b.WriteString("\n\n")

	if len(replies) > 0 {
		b.WriteString("# REPLIES from a previous round:\n\n")
		for agent, reply := range replies {
			b.WriteString(fmt.Sprintf("## Agent %s\n%s\n\n", agent, reply))
		}
	}

	if len(discussion) > 0 {
		b.WriteString("# DISCUSSION\n\n")
		for fromAgent, messages := range discussion {
			b.WriteString(fmt.Sprintf("## With %s\n", fromAgent))
			for _, msg := range messages {
				b.WriteString(fmt.Sprintf("%s\n", msg))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("--- RESPONSE FORMAT ---\n\n")
	b.WriteString("No deviations—strict sections only. Refine # ANSWER using # REPLIES + # DISCUSSION (address gaps/POVs). Always contribute to # DISCUSSION if round >1 and gaps found (1-2 concise messages per relevant agent).\n\n")
	b.WriteString("# ANSWER\n\n")
	b.WriteString("Your refined raw answer (no scaffolding; incorporate priors/discussion; <300 words).\n\n")
	b.WriteString("# RATIONALE\n\n")
	b.WriteString("(Optional) Brief reasoning for changes (e.g., \"Added POV from Grok to fix bias\").\n\n")
	b.WriteString("# DISCUSSION\n\n")
	b.WriteString("## With [AGENT_NAME]\n\n")
	b.WriteString("[Your 1-2 new messages only, e.g., \"Consider environmental counterpoint to your tech focus.\"]\n\n")
	b.WriteString("## With [AGENT_NAME]\n\n")
	b.WriteString("[Same for others if relevant; skip if none]\n\n")
	b.WriteString("Ex for round 2: If replies missed \"ethics,\" # ANSWER weaves it in; # DISCUSSION: \"To Claude: Balance with real-world precedents.\"")

	return b.String()
}

// ParseResponse parses markdown response into Reply struct
func ParseResponse(content string) types.Reply {
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
					saveSection(&reply, currentSection, strings.TrimSpace(sectionContent.String()), currentAgent)
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
		saveSection(&reply, currentSection, strings.TrimSpace(sectionContent.String()), currentAgent)
	}

	return reply
}

// saveSection saves content to the appropriate reply field
func saveSection(reply *types.Reply, section, content, agent string) {
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
