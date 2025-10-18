package shared

import (
	"testing"
)

func TestParseRanking(t *testing.T) {
	content := `# RANKING

Grok
GPT
Claude
Gemini
`

	ranking := ParseRanking(content)

	expected := []string{"Grok", "GPT", "Claude", "Gemini"}
	if len(ranking) != len(expected) {
		t.Fatalf("Expected %d agents, got %d", len(expected), len(ranking))
	}

	for i, agent := range expected {
		if ranking[i] != agent {
			t.Errorf("Position %d: expected %s, got %s", i, agent, ranking[i])
		}
	}
}

func TestAggregateRankings(t *testing.T) {
	rankings := map[string][]string{
		"grok":   {"Grok", "GPT", "Claude"},
		"gpt":    {"GPT", "Grok", "Claude"},
		"claude": {"Grok", "Claude", "GPT"},
	}

	allAgents := []string{"Grok", "GPT", "Claude"}

	winner := AggregateRankings(rankings, allAgents)

	// Grok should win: 3+2+3=8 points
	// GPT: 2+3+1=6 points
	// Claude: 1+1+2=4 points
	if winner != "Grok" {
		t.Errorf("Expected Grok to win, got %s", winner)
	}
}

func TestFormatRankingPrompt(t *testing.T) {
	finalAnswers := map[string]string{
		"Grok":   "Answer from Grok",
		"GPT":    "Answer from GPT",
		"Claude": "Answer from Claude",
	}

	prompt := FormatRankingPrompt("Grok", "What is AI?", []string{"GPT", "Claude"}, finalAnswers)

	if prompt == "" {
		t.Error("Ranking prompt should not be empty")
	}

	// Verify key sections
	tests := []string{
		"You are agent Grok",
		"# QUESTION",
		"# FINAL ANSWERS FROM ALL AGENTS",
		"# RANKING INSTRUCTIONS",
		"# RANKING",
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
