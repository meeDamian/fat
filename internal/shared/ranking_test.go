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

	gold, silver, bronze, _ := AggregateRankings(rankings, allAgents)

	// Grok should win: 3+2+3=8 points
	// GPT: 2+3+1=6 points
	// Claude: 1+1+2=4 points
	if len(gold) != 1 || gold[0] != "Grok" {
		t.Errorf("Expected Grok to win gold, got %v", gold)
	}
	if len(silver) != 1 || silver[0] != "GPT" {
		t.Errorf("Expected GPT as silver, got %v", silver)
	}
	if len(bronze) != 1 || bronze[0] != "Claude" {
		t.Errorf("Expected Claude as bronze, got %v", bronze)
	}
}

func TestAggregateRankingsWithTies(t *testing.T) {
	// Test case where two models tie for gold
	rankings := map[string][]string{
		"grok":   {"Grok", "GPT", "Claude", "Gemini"},
		"gpt":    {"GPT", "Grok", "Claude", "Gemini"},
		"claude": {"Grok", "GPT", "Gemini", "Claude"},
		"gemini": {"GPT", "Grok", "Claude", "Gemini"},
	}

	allAgents := []string{"Grok", "GPT", "Claude", "Gemini"}

	gold, silver, bronze, _ := AggregateRankings(rankings, allAgents)

	// Grok: 4+3+4+3=14 points
	// GPT: 3+4+3+4=14 points (tied for gold!)
	// Claude: 2+2+1+2=7 points
	// Gemini: 1+1+2+1=5 points

	// Both Grok and GPT should have gold
	if len(gold) != 2 {
		t.Errorf("Expected 2 gold winners (tie), got %d: %v", len(gold), gold)
	}

	// Check both are in gold (order may vary)
	hasGrok := false
	hasGPT := false
	for _, winner := range gold {
		if winner == "Grok" {
			hasGrok = true
		}
		if winner == "GPT" {
			hasGPT = true
		}
	}
	if !hasGrok || !hasGPT {
		t.Errorf("Expected both Grok and GPT in gold, got %v", gold)
	}

	// Claude should have silver (next highest score)
	if len(silver) != 1 || silver[0] != "Claude" {
		t.Errorf("Expected Claude as silver, got %v", silver)
	}

	// Gemini should have bronze
	if len(bronze) != 1 || bronze[0] != "Gemini" {
		t.Errorf("Expected Gemini as bronze, got %v", bronze)
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
