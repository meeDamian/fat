package shared

import (
	"fmt"
	"sort"
	"strings"
	
	"github.com/meedamian/fat/internal/types"
)

// FormatRankingPrompt creates a standardized ranking prompt
func FormatRankingPrompt(agentName, question string, otherAgents []string, finalAnswers map[string]types.Reply) string {
	var b strings.Builder

	b.WriteString("╔══════════════════════════════════════════════════════════════╗\n")
	b.WriteString("║               🚨 RANKING MODE - NOT WRITING MODE 🚨          ║\n")
	b.WriteString("║                                                              ║\n")
	b.WriteString("║  YOUR TASK: Judge and rank the answers shown below          ║\n")
	b.WriteString("║  YOUR OUTPUT: A list of agent names, best to worst          ║\n")
	b.WriteString("║                                                              ║\n")
	b.WriteString("║  ❌ DO NOT write a new answer to the question                ║\n")
	b.WriteString("║  ❌ DO NOT use # ANSWER or # RATIONALE sections              ║\n")
	b.WriteString("║  ❌ DO NOT explain your ranking                              ║\n")
	b.WriteString("║                                                              ║\n")
	b.WriteString("║  ✅ ONLY output agent names, one per line                    ║\n")
	b.WriteString("╚══════════════════════════════════════════════════════════════╝\n\n")
	b.WriteString(fmt.Sprintf("You are %s acting as a JUDGE, not as a writer.\n\n", agentName))
	
	b.WriteString("# ORIGINAL QUESTION (for context only - DO NOT answer this)\n\n")
	b.WriteString(question)
	b.WriteString("\n\n")
	
	b.WriteString("# ANSWERS TO RANK\n\n")

	// Sort agent names for consistent ordering
	allAgents := append([]string{agentName}, otherAgents...)
	sort.Strings(allAgents)

	for _, agent := range allAgents {
		if reply, ok := finalAnswers[agent]; ok {
			b.WriteString(fmt.Sprintf("## %s\n\n%s\n\n", agent, reply.Answer))
		}
	}

	b.WriteString("# YOUR TASK\n\n")
	b.WriteString("Evaluate and rank ONLY the answers shown above. Do NOT create a new answer.\n\n")
	b.WriteString("Ranking criteria:\n")
	b.WriteString("- **Accuracy** (40%): Correctness and precision\n")
	b.WriteString("- **Completeness** (30%): Addresses all aspects of the question\n")
	b.WriteString("- **Clarity** (20%): Well-structured and understandable\n")
	b.WriteString("- **Insight** (10%): Depth and originality\n\n")
	b.WriteString("Be objective. You may rank yourself anywhere. Judge on merit, not identity.\n\n")
	
	b.WriteString("═══════════════════════════════════════════════════════════════\n")
	b.WriteString("                    YOUR RESPONSE FORMAT                      \n")
	b.WriteString("═══════════════════════════════════════════════════════════════\n\n")
	b.WriteString("Output ONLY agent names, one per line, ordered from best to worst.\n")
	b.WriteString("NO sections like # ANSWER or # RATIONALE.\n")
	b.WriteString("NO explanations or commentary.\n")
	b.WriteString("JUST the list:\n\n")
	for _, agent := range allAgents {
		b.WriteString(fmt.Sprintf("%s\n", agent))
	}
	b.WriteString("\n(Reorder the above names from best to worst)\n\n")
	b.WriteString("══════════════════════════════════════════════════════════════\n")
	b.WriteString("YOUR RESPONSE MUST BE ONLY AGENT NAMES IN THIS EXACT FORMAT:\n")
	b.WriteString("══════════════════════════════════════════════════════════════\n\n")
	b.WriteString("gpt-5-mini\n")
	b.WriteString("grok-4-fast\n")
	b.WriteString("gemini-2.5-flash\n")
	b.WriteString("claude-3-5-haiku-20241022\n\n")
	b.WriteString("═══════════════════════════════════════════════════════════════\n")
	b.WriteString("NO OTHER TEXT, NO SECTIONS, NO EXPLANATIONS - JUST THE LIST!\n")
	b.WriteString("═══════════════════════════════════════════════════════════════")

	return b.String()
}

// ParseRanking extracts agent names from ranking response
func ParseRanking(content string) []string {
	var ranking []string
	
	// Check if model provided # ANSWER instead of ranking
	hasAnswerSection := strings.Contains(content, "# ANSWER")
	if hasAnswerSection {
		fmt.Printf("DEBUG: Model provided # ANSWER section instead of ranking\n")
		return ranking
	}
	
	hasRankingSection := strings.Contains(content, "# RANKING")
	lines := strings.Split(content, "\n")
	inRankingSection := !hasRankingSection // If no section header, assume whole response is ranking
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Start capturing after # RANKING header if it exists
		if strings.HasPrefix(line, "# RANKING") {
			inRankingSection = true
			continue
		}
		
		// Stop if we hit another section
		if strings.HasPrefix(line, "#") {
			break
		}

		if inRankingSection && line != "" {
			// Skip instruction lines, separators, code blocks
			if strings.Contains(line, "IMPORTANT:") || 
			   strings.Contains(line, "Do NOT") || 
			   strings.Contains(line, "ONLY output") || 
			   strings.Contains(line, "Reorder") ||
			   strings.Contains(line, "one per line") || 
			   strings.Contains(line, "best to worst") ||
			   strings.Contains(line, "YOUR RESPONSE") ||
			   strings.Contains(line, "EXACT FORMAT") ||
			   strings.Contains(line, "NO OTHER TEXT") ||
			   strings.HasPrefix(line, "(") ||
			   strings.HasPrefix(line, "═") ||
			   strings.HasPrefix(line, "╔") ||
			   strings.HasPrefix(line, "╚") ||
			   strings.HasPrefix(line, "║") ||
			   strings.HasPrefix(line, "```") ||
			   strings.HasPrefix(line, "[") ||
			   strings.HasPrefix(line, "]") {
				continue
			}
			
			// Clean up the agent name
			agentName := strings.TrimSpace(line)
			agentName = strings.TrimPrefix(agentName, "Agent ")
			agentName = strings.TrimPrefix(agentName, "- ")
			agentName = strings.TrimPrefix(agentName, "* ")
			agentName = strings.TrimPrefix(agentName, "\"")
			agentName = strings.TrimSuffix(agentName, "\"")
			agentName = strings.TrimSuffix(agentName, ",")
			agentName = strings.TrimSuffix(agentName, ".")
			
			// Accept any non-empty string that doesn't look like instructions
			if agentName != "" && len(agentName) > 2 {
				ranking = append(ranking, agentName)
			}
		}
	}

	return ranking
}

// AggregateRankings combines rankings from multiple agents using Borda count
// Returns winner (1st place) and runnerUp (2nd place)
func AggregateRankings(rankings map[string][]string, allAgents []string) (string, string) {
	scores := make(map[string]int)
	
	// Initialize scores
	for _, agent := range allAgents {
		scores[agent] = 0
	}

	// Borda count: first place gets n points, second gets n-1, etc.
	for rankerID, ranking := range rankings {
		points := len(allAgents)
		fmt.Printf("DEBUG: Processing ranking from %s: %v\n", rankerID, ranking)
		for _, agent := range ranking {
			if _, exists := scores[agent]; exists {
				fmt.Printf("DEBUG: Awarding %d points to %s\n", points, agent)
				scores[agent] += points
				points--
			} else {
				fmt.Printf("DEBUG: Agent %s not in allAgents list!\n", agent)
			}
		}
	}

	// Log all scores before finding winners
	fmt.Printf("DEBUG: Final scores:\n")
	for agent, score := range scores {
		fmt.Printf("DEBUG:   %s: %d points\n", agent, score)
	}

	// Find top 2 winners
	firstScore := -1
	secondScore := -1
	winner := ""
	runnerUp := ""
	
	for agent, score := range scores {
		if score > firstScore {
			// New winner, previous winner becomes runner-up
			secondScore = firstScore
			runnerUp = winner
			firstScore = score
			winner = agent
		} else if score > secondScore && agent != winner {
			// New runner-up
			secondScore = score
			runnerUp = agent
		}
	}

	fmt.Printf("DEBUG: Winner selected: %s with %d points\n", winner, firstScore)
	fmt.Printf("DEBUG: Runner-up selected: %s with %d points\n", runnerUp, secondScore)
	return winner, runnerUp
}
