package shared

import (
	"strings"
	"testing"

	"github.com/meedamian/fat/internal/types"
)

func TestFormatPrompt(t *testing.T) {
	meta := types.Meta{
		Round:       2,
		TotalRounds: 3,
		OtherAgents: []string{"GPT", "Claude"},
	}

	replies := map[string]types.Reply{
		"grok":   {Answer: "Answer from Grok"},
		"gpt":    {Answer: "Answer from GPT"},
		"claude": {Answer: "Answer from Claude"},
	}

	discussion := map[string]map[string][]types.DiscussionMessage{
		"Grok": {
			"GPT": {{From: "Claude", Message: "Consider X", Round: 1}},
		},
	}

	prompt := FormatPrompt("Grok", "What is AI?", meta, replies, discussion)

	// Verify key sections are present
	if !strings.Contains(prompt, "You are Grok in a 3-agent collaboration") {
		t.Error("Missing agent introduction")
	}

	if !strings.Contains(prompt, "Round 2 of 3") {
		t.Error("Missing round information")
	}

	if !strings.Contains(prompt, "# QUESTION") {
		t.Error("Missing question section")
	}

	if !strings.Contains(prompt, "# REPLIES from previous round:") {
		t.Error("Missing replies section")
	}

	if !strings.Contains(prompt, "# DISCUSSION") {
		t.Error("Missing discussion section")
	}

	if !strings.Contains(prompt, "--- RESPONSE FORMAT ---") {
		t.Error("Missing response format section")
	}
	
	if !strings.Contains(prompt, "--- YOUR TASK ---") {
		t.Error("Missing task section")
	}
}

func TestParseResponse(t *testing.T) {
	content := `# ANSWER

This is the answer to the question.

# RATIONALE

This is my reasoning.

# DISCUSSION

## With GPT

Consider adding more context.

## With Claude

Your approach is solid.
`

	reply := ParseResponse(content)

	if reply.Answer != "This is the answer to the question." {
		t.Errorf("Expected answer 'This is the answer to the question.', got '%s'", reply.Answer)
	}

	if reply.Rationale != "This is my reasoning." {
		t.Errorf("Expected rationale 'This is my reasoning.', got '%s'", reply.Rationale)
	}

	if len(reply.Discussion) != 2 {
		t.Errorf("Expected 2 discussion entries, got %d", len(reply.Discussion))
	}

	if reply.Discussion["GPT"] != "Consider adding more context." {
		t.Errorf("Unexpected discussion with GPT: %s", reply.Discussion["GPT"])
	}
}

func TestFormatPromptRound1(t *testing.T) {
	meta := types.Meta{
		Round:       1,
		TotalRounds: 3,
		OtherAgents: []string{"GPT", "Claude"},
	}

	prompt := FormatPrompt("Grok", "Test question", meta, map[string]types.Reply{}, map[string]map[string][]types.DiscussionMessage{})

	// Round 1 should NOT have replies or discussion sections
	if strings.Contains(prompt, "# REPLIES from previous round:") {
		t.Error("Round 1 should not have replies section")
	}

	if strings.Contains(prompt, "(None - this is round 1)") {
		t.Error("Round 1 should not mention 'None - this is round 1'")
	}

	if strings.Contains(prompt, "(No discussion yet)") {
		t.Error("Round 1 should not have discussion placeholder")
	}
	
	if !strings.Contains(prompt, "This is round 1 - provide your initial answer") {
		t.Error("Round 1 should have specific instructions")
	}
	
	if !strings.Contains(prompt, "# RATIONALE") {
		t.Error("Round 1 should have RATIONALE section")
	}
	
	// Round 1 should not have DISCUSSION section in format
	if strings.Contains(prompt, "# DISCUSSION") && strings.Contains(prompt, "## With [AgentName]") {
		t.Error("Round 1 should not have DISCUSSION format section")
	}
}
