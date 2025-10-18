package shared

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"github.com/meedamian/fat/internal/types"
)

// FormatPrompt creates a standardized prompt for all models
func FormatPrompt(modelName, question string, meta types.Meta, replies map[string]string, discussion map[string][]string) string {
	var b strings.Builder

	otherAgentsStr := "none"
	if len(meta.OtherAgents) > 0 {
		otherAgentsStr = strings.Join(meta.OtherAgents, ", ")
	}

	agentCount := len(meta.OtherAgents) + 1
	b.WriteString(fmt.Sprintf("You are %s in a %d-agent collaboration. Other agents: %s. Round %d of %d.\n\n", modelName, agentCount, otherAgentsStr, meta.Round, meta.TotalRounds))
	
	b.WriteString("--- QUESTION ---\n\n")
	b.WriteString(question)
	b.WriteString("\n\n")

	// Context from previous rounds
	if meta.Round > 1 {
		b.WriteString("--- CONTEXT FROM PREVIOUS ROUND ---\n\n")
	}

	b.WriteString("# REPLIES from previous round:\n\n")
	if len(replies) == 0 {
		if meta.Round == 1 {
			b.WriteString("(None - this is round 1)\n\n")
		} else {
			b.WriteString("(No replies available)\n\n")
		}
	} else {
		agentNames := make([]string, 0, len(replies))
		for name := range replies {
			agentNames = append(agentNames, name)
		}
		sort.Strings(agentNames)
		for _, agent := range agentNames {
			answer := strings.TrimSpace(replies[agent])
			if answer == "" {
				answer = "(No answer provided)"
			}
			b.WriteString(fmt.Sprintf("## %s\n%s\n\n", agent, answer))
		}
	}

	b.WriteString("# DISCUSSION\n\n")
	if len(discussion) == 0 {
		if meta.Round == 1 {
			b.WriteString("(No discussion yet)\n\n")
		} else {
			b.WriteString("(No discussion messages)\n\n")
		}
	} else {
		targets := make([]string, 0, len(discussion))
		for target := range discussion {
			targets = append(targets, target)
		}
		sort.Strings(targets)
		for _, target := range targets {
			b.WriteString(fmt.Sprintf("## With %s\n", target))
			msgs := discussion[target]
			for _, msg := range msgs {
				b.WriteString(fmt.Sprintf("%s\n", strings.TrimSpace(msg)))
			}
			b.WriteString("\n")
		}
	}

	// Round-specific instructions
	b.WriteString("--- YOUR TASK ---\n\n")
	if meta.Round == 1 {
		b.WriteString("This is round 1 - provide your initial analysis of the question.\n\n")
		b.WriteString("Focus on:\n")
		b.WriteString("- Answering the question directly and completely\n")
		b.WriteString("- Using your unique perspective and expertise\n")
		b.WriteString("- Being concise but thorough (<300 words)\n\n")
		b.WriteString("Note: No DISCUSSION section needed in round 1.\n\n")
	} else {
		b.WriteString(fmt.Sprintf("This is round %d of %d - refine your answer based on:\n", meta.Round, meta.TotalRounds))
		b.WriteString("1. Gaps or weaknesses in other agents' answers\n")
		b.WriteString("2. Discussion points directed at you\n")
		b.WriteString("3. New perspectives you can contribute\n\n")
		b.WriteString("Refine your ANSWER by:\n")
		b.WriteString("- Incorporating valid points from other agents\n")
		b.WriteString("- Addressing feedback directed at you\n")
		b.WriteString("- Maintaining your core perspective while filling gaps\n")
		b.WriteString("- NOT simply copying other agents' work\n\n")
		b.WriteString("Provide 1-2 discussion messages (20-50 words each) to agents whose answers could benefit from your expertise.\n\n")
	}

	b.WriteString("--- RESPONSE FORMAT ---\n\n")
	b.WriteString("Respond in this EXACT format:\n\n")
	b.WriteString("# ANSWER\n\n")
	b.WriteString("Your refined answer (incorporate feedback, address gaps, <300 words)\n\n")
	
	if meta.Round > 1 {
		b.WriteString("# RATIONALE\n\n")
		b.WriteString("(Optional) Brief explanation of changes made (e.g., \"Added economic data from GPT's suggestion\")\n\n")
	}
	
	if meta.Round > 1 {
		b.WriteString("# DISCUSSION\n\n")
		b.WriteString("(Optional - only if you have substantive feedback)\n\n")
		b.WriteString("## With [AgentName]\n\n")
		b.WriteString("[One specific, actionable suggestion, 20-50 words]\n\n")
		b.WriteString("IMPORTANT RULES:\n")
		b.WriteString("- Omit DISCUSSION section entirely if no substantive feedback\n")
		b.WriteString("- Each message must suggest a specific improvement or ask a clarifying question\n")
		b.WriteString("- Do NOT include prefixes like \"To AgentName:\" - just the message content\n")
		b.WriteString("- Be constructive, not just praise or criticism\n\n")
		b.WriteString("GOOD: \"Your economic analysis omits Q4 2023 inflation data. Adding this would strengthen the GDP impact argument.\"\n\n")
		b.WriteString("BAD: \"Good point!\" or \"I disagree with your approach.\"\n")
	}

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
					// Skip adding heading text to content
					return ast.WalkSkipChildren, nil
				} else if node.Level == 2 && currentSection == "discussion" { // ## headers
					// Save previous discussion entry if exists
					if currentAgent != "" && sectionContent.Len() > 0 {
						saveSection(&reply, currentSection, strings.TrimSpace(sectionContent.String()), currentAgent)
						sectionContent.Reset()
					}
					
					textContent := string(node.Text(reader.Source()))
					if strings.HasPrefix(textContent, "With ") {
						currentAgent = strings.TrimPrefix(textContent, "With ")
					}
					// Skip adding heading text to content
					return ast.WalkSkipChildren, nil
				}
			case *ast.Text:
				if currentSection != "" && node.Parent().Kind() != ast.KindHeading {
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
