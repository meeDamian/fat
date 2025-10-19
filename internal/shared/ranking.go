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

	b.WriteString(fmt.Sprintf("You are %s. Rank all answers objectively based on quality.\n\n", agentName))
	
	b.WriteString("# QUESTION\n\n")
	b.WriteString(question)
	b.WriteString("\n\n")
	
	b.WriteString("# ANSWERS\n\n")

	// Sort agent names for consistent ordering
	allAgents := append([]string{agentName}, otherAgents...)
	sort.Strings(allAgents)

	for _, agent := range allAgents {
		if reply, ok := finalAnswers[agent]; ok {
			b.WriteString(fmt.Sprintf("## %s\n\n%s\n\n", agent, reply.Answer))
		}
	}

	b.WriteString("# YOUR TASK\n\n")
	b.WriteString("Rank the above answers from best to worst based on:\n\n")
	b.WriteString("- **Accuracy** (40%): Correctness and precision\n")
	b.WriteString("- **Completeness** (30%): Addresses all aspects of the question\n")
	b.WriteString("- **Clarity** (20%): Well-structured and understandable\n")
	b.WriteString("- **Insight** (10%): Depth and originality\n\n")
	b.WriteString("Be objective. You may rank yourself anywhere. Judge on merit, not identity.\n\n")
	
	b.WriteString("# RANKING\n\n")
	b.WriteString("Output ONLY agent names, one per line, best to worst:\n\n")
	for _, agent := range allAgents {
		b.WriteString(fmt.Sprintf("%s\n", agent))
	}
	b.WriteString("\n(Reorder above from best to worst)")

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
