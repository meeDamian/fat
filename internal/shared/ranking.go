package shared

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	
	"github.com/meedamian/fat/internal/types"
)

// FormatRankingPrompt creates a standardized ranking prompt with anonymized agents
func FormatRankingPrompt(agentName, question string, otherAgents []string, finalAnswers map[string]types.Reply) string {
	var b strings.Builder

	// Create anonymization mapping
	allAgents := append([]string{agentName}, otherAgents...)
	sort.Strings(allAgents) // Sort for consistency
	
	// Generate random letter assignments
	letters := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
	rand.Shuffle(len(letters), func(i, j int) { letters[i], letters[j] = letters[j], letters[i] })
	
	// Map real names to anonymous letters
	anonMap := make(map[string]string)
	reverseMap := make(map[string]string)
	for i, agent := range allAgents {
		letter := letters[i]
		anonMap[agent] = letter
		reverseMap[letter] = agent
	}

	b.WriteString("╔══════════════════════════════════════════════════════════════╗\n")
	b.WriteString("║               🚨 RANKING MODE - NOT WRITING MODE 🚨          ║\n")
	b.WriteString("║                                                              ║\n")
	b.WriteString("║  YOUR TASK: Judge and rank the answers shown below          ║\n")
	b.WriteString("║  YOUR OUTPUT: A list of agent letters, best to worst        ║\n")
	b.WriteString("║                                                              ║\n")
	b.WriteString("║  ❌ DO NOT write a new answer to the question                ║\n")
	b.WriteString("║  ❌ DO NOT use # ANSWER or # RATIONALE sections              ║\n")
	b.WriteString("║  ❌ DO NOT explain your ranking                              ║\n")
	b.WriteString("║                                                              ║\n")
	b.WriteString("║  ✅ ONLY output agent letters, one per line                  ║\n")
	b.WriteString("╚══════════════════════════════════════════════════════════════╝\n\n")
	b.WriteString("You are acting as a JUDGE, not as a writer.\n\n")
	
	b.WriteString("# ORIGINAL QUESTION (for context only - DO NOT answer this)\n\n")
	b.WriteString(question)
	b.WriteString("\n\n")
	
	b.WriteString("# ANSWERS TO RANK\n\n")

	// Show answers with anonymous letters
	for _, agent := range allAgents {
		if reply, ok := finalAnswers[agent]; ok {
			letter := anonMap[agent]
			b.WriteString(fmt.Sprintf("## Agent %s\n\n%s\n\n", letter, reply.Answer))
		}
	}

	b.WriteString("# YOUR TASK\n\n")
	b.WriteString("Evaluate and rank ONLY the answers shown above. Do NOT create a new answer.\n\n")
	b.WriteString("═══════════════════════════════════════════════════════════════\n")
	b.WriteString("                    ⚠️  CRITICAL REQUIREMENT  ⚠️                \n")
	b.WriteString("═══════════════════════════════════════════════════════════════\n\n")
	b.WriteString("**PROMPT ADHERENCE IS MANDATORY**\n\n")
	b.WriteString("If the original question specifies format requirements (word count, length,\n")
	b.WriteString("structure, style, etc.), answers that violate these requirements MUST be\n")
	b.WriteString("ranked significantly lower, regardless of content quality.\n\n")
	b.WriteString("Examples of violations:\n")
	b.WriteString("- Question asks for \"5 words\" → Answer provides 4, 6, or a paragraph\n")
	b.WriteString("- Question asks for \"one sentence\" → Answer provides multiple sentences\n")
	b.WriteString("- Question asks for \"bullet points\" → Answer provides prose\n\n")
	b.WriteString("Prompt adherence violations should result in severe ranking penalties.\n\n")
	b.WriteString("═══════════════════════════════════════════════════════════════\n\n")
	b.WriteString("Ranking criteria (for answers that follow the prompt):\n")
	b.WriteString("- **Accuracy** (40%): Correctness and precision\n")
	b.WriteString("- **Completeness** (30%): Addresses all aspects of the question\n")
	b.WriteString("- **Clarity** (20%): Well-structured and understandable\n")
	b.WriteString("- **Insight** (10%): Depth and originality\n\n")
	b.WriteString("Be objective. Judge on merit, not identity.\n\n")
	
	b.WriteString("═══════════════════════════════════════════════════════════════\n")
	b.WriteString("                    YOUR RESPONSE FORMAT                      \n")
	b.WriteString("═══════════════════════════════════════════════════════════════\n\n")
	b.WriteString("Output ONLY agent letters, one per line, ordered from best to worst.\n")
	b.WriteString("NO sections like # ANSWER or # RATIONALE.\n")
	b.WriteString("NO explanations or commentary.\n")
	b.WriteString("JUST the list:\n\n")
	
	// Show example with the anonymous letters
	for _, agent := range allAgents {
		b.WriteString(fmt.Sprintf("%s\n", anonMap[agent]))
	}
	b.WriteString("\n(Reorder the above letters from best to worst)\n\n")
	b.WriteString("══════════════════════════════════════════════════════════════\n")
	b.WriteString("YOUR RESPONSE MUST BE ONLY AGENT LETTERS IN THIS EXACT FORMAT:\n")
	b.WriteString("══════════════════════════════════════════════════════════════\n\n")
	
	// Show example ranking with letters
	exampleLetters := make([]string, len(allAgents))
	copy(exampleLetters, letters[:len(allAgents)])
	for _, letter := range exampleLetters {
		b.WriteString(fmt.Sprintf("%s\n", letter))
	}
	
	b.WriteString("\n═══════════════════════════════════════════════════════════════\n")
	b.WriteString("NO OTHER TEXT, NO SECTIONS, NO EXPLANATIONS - JUST THE LIST!\n")
	b.WriteString("═══════════════════════════════════════════════════════════════\n\n")
	
	// Add mapping at the end for the system to decode (hidden from model's perspective in practice)
	b.WriteString("<!-- ANONYMIZATION_MAP:")
	for letter, agent := range reverseMap {
		b.WriteString(fmt.Sprintf(" %s=%s", letter, agent))
	}
	b.WriteString(" -->")

	return b.String()
}

// ParseRanking extracts agent letters from ranking response and decodes them using the prompt's mapping
func ParseRanking(content string, prompt string) []string {
	var ranking []string
	
	// Extract anonymization mapping from prompt
	letterToAgent := extractAnonymizationMap(prompt)
	
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
			
			// Clean up the letter/agent name
			agentName := strings.TrimSpace(line)
			agentName, _ = strings.CutPrefix(agentName, "Agent ")
			agentName, _ = strings.CutPrefix(agentName, "- ")
			agentName, _ = strings.CutPrefix(agentName, "* ")
			agentName, _ = strings.CutPrefix(agentName, "\"")
			agentName = strings.TrimSuffix(agentName, "\"")
			agentName = strings.TrimSuffix(agentName, ",")
			agentName = strings.TrimSuffix(agentName, ".")
			
			// Check if it's a single letter (anonymized)
			if len(agentName) == 1 && agentName >= "A" && agentName <= "H" {
				// Decode the letter to real agent name
				if realName, ok := letterToAgent[agentName]; ok {
					ranking = append(ranking, realName)
				} else {
					fmt.Printf("DEBUG: Unknown letter %s in ranking\n", agentName)
				}
			} else if agentName != "" && len(agentName) > 2 {
				// Fallback: accept full agent names (for backwards compatibility)
				ranking = append(ranking, agentName)
			}
		}
	}

	return ranking
}

// extractAnonymizationMap extracts the letter-to-agent mapping from the prompt
func extractAnonymizationMap(prompt string) map[string]string {
	mapping := make(map[string]string)
	
	// Find the mapping comment in the prompt
	startIdx := strings.Index(prompt, "<!-- ANONYMIZATION_MAP:")
	if startIdx == -1 {
		return mapping
	}
	
	endIdx := strings.Index(prompt[startIdx:], "-->")
	if endIdx == -1 {
		return mapping
	}
	
	mapStr := prompt[startIdx+len("<!-- ANONYMIZATION_MAP:") : startIdx+endIdx]
	pairs := strings.Fields(mapStr)
	
	for _, pair := range pairs {
		parts := strings.Split(pair, "=")
		if len(parts) == 2 {
			letter := strings.TrimSpace(parts[0])
			agent := strings.TrimSpace(parts[1])
			mapping[letter] = agent
		}
	}
	
	return mapping
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
