package shared

import (
	"fmt"
	"sort"
	"strings"
)

// FormatRankingPrompt creates a standardized ranking prompt
func FormatRankingPrompt(agentName, question string, otherAgents []string, finalAnswers map[string]string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("You are %s. Your task is to objectively rank the quality of all answers (including your own) for the question below.\n\n", agentName))
	b.WriteString("# QUESTION\n\n")
	b.WriteString(question)
	b.WriteString("\n\n")
	b.WriteString("# FINAL ANSWERS FROM ALL AGENTS\n\n")

	// Sort agent names for consistent ordering
	allAgents := append([]string{agentName}, otherAgents...)
	sort.Strings(allAgents)

	for _, agent := range allAgents {
		if answer, ok := finalAnswers[agent]; ok {
			b.WriteString(fmt.Sprintf("## %s\n%s\n\n", agent, answer))
		}
	}

	b.WriteString("# RANKING CRITERIA\n\n")
	b.WriteString("Rank ALL agents from best to worst based on these weighted criteria:\n\n")
	b.WriteString("1. **Factual Accuracy** (40%) - Correctness and precision of information\n")
	b.WriteString("2. **Completeness** (30%) - Addresses all aspects of the question\n")
	b.WriteString("3. **Clarity and Coherence** (20%) - Well-structured and easy to understand\n")
	b.WriteString("4. **Integration of Discussion** (10%) - Incorporated feedback from collaboration\n\n")
	b.WriteString("IMPORTANT:\n")
	b.WriteString("- Be objective - you may rank yourself anywhere based on these criteria\n")
	b.WriteString("- Consider the full collaboration, not just the final answer\n")
	b.WriteString("- Rank based on merit, not agent identity\n\n")
	
	b.WriteString("# OUTPUT FORMAT\n\n")
	b.WriteString("Output ONLY the ranking in this exact format (one agent name per line, best first):\n\n")
	b.WriteString("# RANKING\n\n")
	for _, agent := range allAgents {
		b.WriteString(fmt.Sprintf("%s\n", agent))
	}
	b.WriteString("\n(Reorder the above names from best to worst)")

	return b.String()
}

// ParseRanking extracts agent names from ranking response
func ParseRanking(content string) []string {
	var ranking []string
	inRankingSection := false

	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.HasPrefix(line, "# RANKING") {
			inRankingSection = true
			continue
		}

		if inRankingSection && line != "" && !strings.HasPrefix(line, "#") {
			// Clean up the agent name
			agentName := strings.TrimSpace(line)
			agentName = strings.TrimPrefix(agentName, "Agent ")
			agentName = strings.TrimPrefix(agentName, "- ")
			agentName = strings.TrimSuffix(agentName, ".")
			
			if agentName != "" {
				ranking = append(ranking, agentName)
			}
		}
	}

	return ranking
}

// AggregateRankings combines rankings from multiple agents using Borda count
func AggregateRankings(rankings map[string][]string, allAgents []string) string {
	scores := make(map[string]int)
	
	// Initialize scores
	for _, agent := range allAgents {
		scores[agent] = 0
	}

	// Borda count: first place gets n points, second gets n-1, etc.
	for _, ranking := range rankings {
		points := len(allAgents)
		for _, agent := range ranking {
			if _, exists := scores[agent]; exists {
				scores[agent] += points
				points--
			}
		}
	}

	// Find winner
	maxScore := -1
	winner := ""
	for agent, score := range scores {
		if score > maxScore {
			maxScore = score
			winner = agent
		}
	}

	return winner
}
