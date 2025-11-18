package shared

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/meedamian/fat/internal/types"
)

// FormatPrompt creates a standardized prompt for all models
// modelID is the short ID (e.g., "grok", "claude") used for discussion lookup
// modelName is the full name (e.g., "grok-4-fast") used for display
func FormatPrompt(modelID, modelName, question string, meta types.Meta, replies map[string]types.Reply, discussion map[string]map[string][]types.DiscussionMessage) string {
	var b strings.Builder

	otherAgentsStr := "none"
	if len(meta.OtherAgents) > 0 {
		otherAgentsStr = strings.Join(meta.OtherAgents, ", ")
	}

	agentCount := len(meta.OtherAgents) + 1
	b.WriteString(fmt.Sprintf("You are %s in a %d-agent collaboration. Other agents: %s. Round %d of %d.\n\n", modelName, agentCount, otherAgentsStr, meta.Round, meta.TotalRounds))

	b.WriteString("# QUESTION\n\n")
	b.WriteString(question)
	b.WriteString("\n\n")

	// Only show context from previous rounds if not round 1
	if meta.Round > 1 {
		b.WriteString("# REPLIES from previous round:\n\n")
		if len(replies) == 0 {
			b.WriteString("(No replies available)\n\n")
		} else {
			// Show own previous answer first (replies map uses modelID as key)
			if ownReply, hasOwn := replies[modelID]; hasOwn {
				answer := strings.TrimSpace(ownReply.Answer)
				if answer == "" {
					answer = "(No answer provided)"
				}
				b.WriteString(fmt.Sprintf("## Your previous answer (%s)\n\n", modelName))
				b.WriteString(answer)
				b.WriteString("\n\n")

				// Include rationale if provided
				if strings.TrimSpace(ownReply.Rationale) != "" {
					b.WriteString(fmt.Sprintf("### Rationale\n\n%s\n\n", strings.TrimSpace(ownReply.Rationale)))
				}
			}

			// Show other agents' answers
			agentIDs := make([]string, 0, len(replies))
			for agentID := range replies {
				if agentID != modelID {
					agentIDs = append(agentIDs, agentID)
				}
			}
			sort.Strings(agentIDs)

			// Map short IDs to display names
			idToDisplayName := map[string]string{
				"grok":   "Grok",
				"gpt":    "GPT",
				"claude": "Claude",
				"gemini": "Gemini",
			}

			// Build a map of agentID -> full model name from OtherAgents
			agentIDToFullName := make(map[string]string)
			for _, fullName := range meta.OtherAgents {
				// Match by checking if the full name contains the agent ID pattern
				lowerFullName := strings.ToLower(fullName)
				if strings.Contains(lowerFullName, "grok") {
					agentIDToFullName["grok"] = fullName
				} else if strings.Contains(lowerFullName, "gpt") {
					agentIDToFullName["gpt"] = fullName
				} else if strings.Contains(lowerFullName, "claude") {
					agentIDToFullName["claude"] = fullName
				} else if strings.Contains(lowerFullName, "gemini") {
					agentIDToFullName["gemini"] = fullName
				}
			}

			for _, agentID := range agentIDs {
				reply := replies[agentID]
				answer := strings.TrimSpace(reply.Answer)
				if answer == "" {
					answer = "(No answer provided)"
				}

				// Get display name for this agent
				displayName := idToDisplayName[agentID]
				if displayName == "" {
					displayName = agentID
				}

				// Get full model name
				fullModelName := agentIDToFullName[agentID]
				if fullModelName == "" {
					fullModelName = agentID
				}

				b.WriteString(fmt.Sprintf("## %s (%s)\n\n%s\n\n", displayName, fullModelName, answer))

				// Include rationale if provided
				if strings.TrimSpace(reply.Rationale) != "" {
					b.WriteString(fmt.Sprintf("### Rationale\n\n%s\n\n", strings.TrimSpace(reply.Rationale)))
				}
			}
		}

		// Show conversation threads with other agents
		if threads, hasThreads := discussion[modelID]; hasThreads && len(threads) > 0 {
			// Check if there are any threads with messages
			hasContent := false
			for _, messages := range threads {
				if len(messages) > 0 {
					hasContent = true
					break
				}
			}

			if hasContent {
				b.WriteString("# DISCUSSION\n\n")

				// Sort agent names for consistent ordering
				agents := make([]string, 0, len(threads))
				for agent := range threads {
					if len(threads[agent]) > 0 {
						agents = append(agents, agent)
					}
				}
				sort.Strings(agents)

				for _, agent := range agents {
					messages := threads[agent]
					if len(messages) == 0 {
						continue
					}

					b.WriteString(fmt.Sprintf("## With %s\n\n", agent))

					// Find the latest message from each party
					var lastFromMe, lastToMe *types.DiscussionMessage
					for i := len(messages) - 1; i >= 0; i-- {
						msg := &messages[i]
						if msg.From == modelID && lastFromMe == nil {
							lastFromMe = msg
						} else if msg.From == agent && lastToMe == nil {
							lastToMe = msg
						}
						// Stop once we have both (or confirmed we don't have one)
						if lastFromMe != nil && lastToMe != nil {
							break
						}
					}

					// Show context: my last message to them (if any)
					if lastFromMe != nil {
						trimmed := strings.TrimSpace(lastFromMe.Message)
						if trimmed != "" {
							b.WriteString(fmt.Sprintf("%s: %s\n\n", lastFromMe.From, trimmed))
						}
					}

					// Show the latest message from them to me
					if lastToMe != nil {
						trimmed := strings.TrimSpace(lastToMe.Message)
						if trimmed != "" {
							b.WriteString(fmt.Sprintf("%s: %s\n\n", lastToMe.From, trimmed))
						}
					}
				}
			}
		}
	}

	// Round-specific instructions
	b.WriteString("--- YOUR TASK ---\n\n")
	if meta.Round == 1 {
		b.WriteString("This is round 1 - provide your initial answer to the question.\n\n")
		b.WriteString("Focus on:\n")
		b.WriteString("- Answering the question directly and completely\n")
		b.WriteString("- Using your unique perspective and expertise\n")
		b.WriteString("- Being concise but thorough (<300 words)\n\n")
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
		b.WriteString("In DISCUSSION messages:\n")
		b.WriteString("- Point out logical flaws, contradictions, or reasoning errors\n")
		b.WriteString("- Challenge assumptions that don't align with the question context\n")
		b.WriteString("- Identify when agents suggest things already present in the discussion\n")
		b.WriteString("- Flag violations of the original prompt's requirements (format, length, structure, etc.)\n")
		b.WriteString("- Provide 1-2 specific, actionable messages (20-50 words each)\n\n")
	}

	b.WriteString("--- RESPONSE FORMAT ---\n\n")
	b.WriteString("Respond in this EXACT format:\n\n")
	b.WriteString("# ANSWER\n\n")
	if meta.Round == 1 {
		b.WriteString("Your answer to the question (<300 words)\n")
		b.WriteString("IMPORTANT: Include ONLY the raw answer here - no scaffolding, disclaimers, or meta-commentary.\n")
		b.WriteString("Save explanations for the RATIONALE section.\n\n")
	} else {
		b.WriteString("Your refined answer (incorporate feedback, address gaps, <300 words)\n")
		b.WriteString("IMPORTANT: Include ONLY the raw answer here - no scaffolding, disclaimers, or meta-commentary.\n")
		b.WriteString("Save explanations for the RATIONALE section.\n\n")
	}

	b.WriteString("# RATIONALE\n\n")
	if meta.Round == 1 {
		b.WriteString("(Optional) Brief explanation of your approach or reasoning\n")
		b.WriteString("⚠️  Use EXACTLY '# RATIONALE' (single #), NOT '### Rationale' or any other format\n\n")
	} else {
		b.WriteString("(Optional) Brief explanation of changes made (e.g., \"Added economic data from GPT's suggestion\")\n")
		b.WriteString("⚠️  Use EXACTLY '# RATIONALE' (single #), NOT '### Rationale' or any other format\n\n")
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

// extractContentFromJSON attempts to extract text content from JSON responses
// Some reasoning models (like Mistral's magistral) return JSON with thinking/content fields
func extractContentFromJSON(content string) string {
	trimmed := strings.TrimSpace(content)

	// Check if content looks like JSON (starts with [ or {)
	if !strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "{") {
		return content
	}

	// Try to parse as JSON array
	var jsonArray []map[string]any
	if err := json.Unmarshal([]byte(trimmed), &jsonArray); err == nil {
		// Extract content from JSON array
		var extracted strings.Builder
		for _, item := range jsonArray {
			// Look for "content" or "text" fields
			if contentVal, ok := item["content"].(string); ok && contentVal != "" {
				extracted.WriteString(contentVal)
				extracted.WriteString("\n")
			} else if textVal, ok := item["text"].(string); ok && textVal != "" {
				extracted.WriteString(textVal)
				extracted.WriteString("\n")
			}
		}
		if extracted.Len() > 0 {
			return strings.TrimSpace(extracted.String())
		}
	}

	// Try to parse as JSON object
	var jsonObj map[string]any
	if err := json.Unmarshal([]byte(trimmed), &jsonObj); err == nil {
		// Look for common content fields
		if contentVal, ok := jsonObj["content"].(string); ok && contentVal != "" {
			return contentVal
		}
		if textVal, ok := jsonObj["text"].(string); ok && textVal != "" {
			return textVal
		}
		if answerVal, ok := jsonObj["answer"].(string); ok && answerVal != "" {
			return answerVal
		}
	}

	// If JSON parsing failed or no content found, return original
	return content
}

// ParseResponse parses markdown response into Reply struct
// Preserves original formatting including list markers, indentation, and blank lines
func ParseResponse(content string) types.Reply {
	reply := types.Reply{
		Discussion: make(map[string]string),
		RawContent: content,
	}

	// Handle JSON responses from thinking/reasoning models
	content = extractContentFromJSON(content)

	lines := strings.Split(content, "\n")
	var currentSection string
	var currentAgent string
	var sectionLines []string
	foundAnySection := false

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Check for # ANSWER, # RATIONALE, # DISCUSSION headings
		// Also handle common mistakes like ### Rationale
		if strings.HasPrefix(trimmed, "# ") {
			// Save previous section
			if currentSection != "" {
				saveSection(&reply, currentSection, strings.Join(sectionLines, "\n"), currentAgent)
				sectionLines = nil
				currentAgent = ""
			}

			heading := strings.TrimSpace(trimmed[2:])
			switch heading {
			case "ANSWER":
				currentSection = "answer"
				foundAnySection = true
			case "RATIONALE":
				currentSection = "rationale"
				foundAnySection = true
			case "DISCUSSION":
				currentSection = "discussion"
				foundAnySection = true
			default:
				currentSection = ""
			}
			continue
		}

		// Handle common formatting mistakes: ### Rationale, ### Answer, etc.
		if strings.HasPrefix(trimmed, "### ") {
			heading := strings.ToUpper(strings.TrimSpace(trimmed[4:]))
			if heading == "RATIONALE" || heading == "ANSWER" || heading == "DISCUSSION" {
				// Save previous section
				if currentSection != "" {
					saveSection(&reply, currentSection, strings.Join(sectionLines, "\n"), currentAgent)
					sectionLines = nil
					currentAgent = ""
				}

				foundAnySection = true
				switch heading {
				case "ANSWER":
					currentSection = "answer"
				case "RATIONALE":
					currentSection = "rationale"
				case "DISCUSSION":
					currentSection = "discussion"
				}
				continue
			}
		}

		// Check for ## With AgentName in discussion section
		if currentSection == "discussion" && strings.HasPrefix(trimmed, "## ") {
			// Save previous discussion entry
			if currentAgent != "" {
				saveSection(&reply, currentSection, strings.Join(sectionLines, "\n"), currentAgent)
				sectionLines = nil
			}

			heading := strings.TrimSpace(trimmed[3:])
			if agent, found := strings.CutPrefix(heading, "With "); found {
				currentAgent = agent
			}
			continue
		}

		// Accumulate content for current section
		if currentSection != "" {
			sectionLines = append(sectionLines, line)
		}
	}

	// Save final section
	if currentSection != "" {
		saveSection(&reply, currentSection, strings.Join(sectionLines, "\n"), currentAgent)
	}

	// If no section headers were found at all, treat entire response as rationale
	// This handles cases where models refuse to follow format
	if !foundAnySection {
		trimmedContent := strings.TrimSpace(content)
		if trimmedContent != "" {
			reply.Rationale = trimmedContent
		}
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
