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

	b.WriteString("CRITICAL: This is a RANKING task, NOT a question-answering task. Do NOT provide a new answer.\n\n")
	b.WriteString(fmt.Sprintf("You are %s. Your ONLY job is to rank the existing answers below.\n\n", agentName))
	
	b.WriteString("# QUESTION\n\n")
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
	
	b.WriteString("# RANKING\n\n")
	b.WriteString("IMPORTANT: Output ONLY the section below with agent names reordered from best to worst.\n")
	b.WriteString("Do NOT include # ANSWER, # RATIONALE, or any other sections.\n")
	b.WriteString("Do NOT provide explanations or commentary.\n")
	b.WriteString("ONLY output agent names, one per line:\n\n")
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
		
		// Start capturing after # RANKING header
		if strings.HasPrefix(line, "# RANKING") {
			inRankingSection = true
			continue
		}
		
		// Stop if we hit another section (like # ANSWER or # RATIONALE)
		if inRankingSection && strings.HasPrefix(line, "#") {
			break
		}

		if inRankingSection && line != "" {
			// Skip instruction lines
			if strings.Contains(line, "IMPORTANT:") || strings.Contains(line, "Do NOT") || 
			   strings.Contains(line, "ONLY output") || strings.Contains(line, "Reorder") ||
			   strings.Contains(line, "one per line") || strings.Contains(line, "best to worst") ||
			   strings.HasPrefix(line, "(") {
				continue
			}
			
			// Clean up the agent name
			agentName := strings.TrimSpace(line)
			agentName = strings.TrimPrefix(agentName, "Agent ")
			agentName = strings.TrimPrefix(agentName, "- ")
			agentName = strings.TrimPrefix(agentName, "* ")
			agentName = strings.TrimPrefix(agentName, "1. ")
			agentName = strings.TrimPrefix(agentName, "2. ")
			agentName = strings.TrimPrefix(agentName, "3. ")
			agentName = strings.TrimPrefix(agentName, "4. ")
			agentName = strings.TrimSuffix(agentName, ".")
			agentName = strings.TrimSuffix(agentName, ",")
			
			// Accept any non-empty string that doesn't look like instructions
			if agentName != "" && len(agentName) > 2 {
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
