package shared

import (
	"testing"
	
	"github.com/meedamian/fat/internal/types"
)

func TestParseRanking(t *testing.T) {
	// Test with anonymized letters
	prompt := `<!-- ANONYMIZATION_MAP: A=Grok B=GPT C=Claude D=Gemini -->`
	content := `# RANKING

A
B
C
D
`

	ranking := ParseRanking(content, prompt)

	expected := []string{"Grok", "GPT", "Claude", "Gemini"}
	if len(ranking) != len(expected) {
		t.Fatalf("Expected %d agents, got %d", len(expected), len(ranking))
	}

	for i, agent := range expected {
		if ranking[i] != agent {
			t.Errorf("Position %d: expected %s, got %s", i, agent, ranking[i])
		}
	}
	
	// Test backwards compatibility with full names
	contentFullNames := `# RANKING

Grok
GPT
Claude
Gemini
`
	rankingFullNames := ParseRanking(contentFullNames, "")
	if len(rankingFullNames) != len(expected) {
		t.Fatalf("Expected %d agents with full names, got %d", len(expected), len(rankingFullNames))
	}
}

func TestAggregateRankings(t *testing.T) {
	rankings := map[string][]string{
		"grok":   {"Grok", "GPT", "Claude"},
		"gpt":    {"GPT", "Grok", "Claude"},
		"claude": {"Grok", "Claude", "GPT"},
	}

	allAgents := []string{"Grok", "GPT", "Claude"}

	winner, runnerUp := AggregateRankings(rankings, allAgents)

	// Grok should win: 3+2+3=8 points
	// GPT: 2+3+1=6 points
	// Claude: 1+1+2=4 points
	if winner != "Grok" {
		t.Errorf("Expected Grok to win, got %s", winner)
	}
	if runnerUp != "GPT" {
		t.Errorf("Expected GPT as runner-up, got %s", runnerUp)
	}
}

func TestFormatRankingPrompt(t *testing.T) {
	finalAnswers := map[string]types.Reply{
		"Grok":   {Answer: "Answer from Grok"},
		"GPT":    {Answer: "Answer from GPT"},
		"Claude": {Answer: "Answer from Claude"},
	}
	
	costs := map[string]float64{
		"Grok":   0.001,
		"GPT":    0.002,
		"Claude": 0.0015,
	}
	
	allAgents := []string{"Grok", "GPT", "Claude"}
	anonMap := CreateAnonymizationMap(allAgents)

	prompt := FormatRankingPrompt("Grok", "What is AI?", []string{"GPT", "Claude"}, finalAnswers, anonMap, costs)

	if prompt == "" {
		t.Error("Ranking prompt should not be empty")
	}

	// Verify key sections
	tests := []string{
		"acting as a JUDGE",
		"ORIGINAL QUESTION",
		"# ANSWERS TO RANK",
		"# YOUR TASK",
		"YOUR RESPONSE FORMAT",
		"Accuracy",
		"Completeness",
		"Clarity",
		"Insight",
		"RANKING MODE",
	}

	for _, test := range tests {
		if !contains(prompt, test) {
			t.Errorf("Ranking prompt missing: %s", test)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
