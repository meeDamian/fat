package shared

import (
	"strings"
	"testing"

	"github.com/meedamian/fat/internal/types"
)

// TestFormatPrompt verifies that prompts include all required sections for multi-round collaboration
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

	prompt := FormatPrompt("grok", "Grok", "What is AI?", meta, replies, discussion)

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

// TestParseResponse verifies basic parsing of ANSWER, RATIONALE, and DISCUSSION sections
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

// TestFormatPromptRound1 verifies that round 1 prompts exclude replies/discussion sections
func TestFormatPromptRound1(t *testing.T) {
	meta := types.Meta{
		Round:       1,
		TotalRounds: 3,
		OtherAgents: []string{"GPT", "Claude"},
	}

	prompt := FormatPrompt("grok", "Grok", "Test question", meta, map[string]types.Reply{}, map[string]map[string][]types.DiscussionMessage{})

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

// TestParseResponse_NumberedList verifies that numbered lists preserve their markers (1., 1), etc.)
func TestParseResponse_NumberedList(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "numbered list with periods",
			content: `# ANSWER

1. Ukraine
2. Philippines
3. Colombia

# RATIONALE

Test rationale.`,
			expected: "1. Ukraine\n2. Philippines\n3. Colombia",
		},
		{
			name: "numbered list with parentheses",
			content: `# ANSWER

1) First item
2) Second item
3) Third item

# RATIONALE

Test.`,
			expected: "1) First item\n2) Second item\n3) Third item",
		},
		{
			name: "numbered list with trailing spaces (hard breaks)",
			content: `# ANSWER

1. Ukraine  
2. Philippines  
3. Colombia  

# RATIONALE

Test.`,
			expected: "1. Ukraine  \n2. Philippines  \n3. Colombia",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reply := ParseResponse(tt.content)
			if reply.Answer != tt.expected {
				t.Errorf("Expected answer %q, got %q", tt.expected, reply.Answer)
			}
		})
	}
}

// TestParseResponse_BulletedLists verifies that bullet markers (-, *, •) are preserved exactly
func TestParseResponse_BulletedLists(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "hyphen bullets",
			content: `# ANSWER

- Philippines
- Colombia
- Brazil

# RATIONALE

Test.`,
			expected: "- Philippines\n- Colombia\n- Brazil",
		},
		{
			name: "asterisk bullets",
			content: `# ANSWER

* Philippines
* Ukraine
* Colombia

# RATIONALE

Test.`,
			expected: "* Philippines\n* Ukraine\n* Colombia",
		},
		{
			name: "unicode bullets",
			content: `# ANSWER

• First
• Second
• Third

# RATIONALE

Test.`,
			expected: "• First\n• Second\n• Third",
		},
		{
			name: "mixed bullet styles",
			content: `# ANSWER

- Item one
* Item two
- Item three

# RATIONALE

Test.`,
			expected: "- Item one\n* Item two\n- Item three",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reply := ParseResponse(tt.content)
			if reply.Answer != tt.expected {
				t.Errorf("Expected answer %q, got %q", tt.expected, reply.Answer)
			}
		})
	}
}

// TestParseResponse_PlainLines verifies that plain text with newlines and blank lines is preserved
func TestParseResponse_PlainLines(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "plain multiline text",
			content: `# ANSWER

Philippines
Ukraine
Russia

# RATIONALE

Test.`,
			expected: "Philippines\nUkraine\nRussia",
		},
		{
			name: "plain text with blank lines",
			content: `# ANSWER

First paragraph.

Second paragraph.

# RATIONALE

Test.`,
			expected: "First paragraph.\n\nSecond paragraph.",
		},
		{
			name: "single line answer",
			content: `# ANSWER

Just one line

# RATIONALE

Test.`,
			expected: "Just one line",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reply := ParseResponse(tt.content)
			if reply.Answer != tt.expected {
				t.Errorf("Expected answer %q, got %q", tt.expected, reply.Answer)
			}
		})
	}
}

// TestParseResponse_MissingAnswer verifies handling of responses without ANSWER sections
func TestParseResponse_MissingAnswer(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "no answer section",
			content: `# RATIONALE

Just rationale here.`,
			expected: "",
		},
		{
			name: "empty answer section",
			content: `# ANSWER

# RATIONALE

Test.`,
			expected: "",
		},
		{
			name: "refusal to answer",
			content: `I do not feel comfortable providing recommendations about obtaining a wife from specific countries.`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reply := ParseResponse(tt.content)
			if reply.Answer != tt.expected {
				t.Errorf("Expected answer %q, got %q", tt.expected, reply.Answer)
			}
			// For the refusal case, verify it goes into rationale
			if tt.name == "refusal to answer" {
				if reply.Rationale == "" {
					t.Error("Unformatted refusal should be captured in rationale")
				}
				if reply.Rationale != tt.content {
					t.Errorf("Expected rationale to be %q, got %q", tt.content, reply.Rationale)
				}
			}
		})
	}
}

// TestParseResponse_ComplexFormatting verifies preservation of nested lists, inline code, and indentation
func TestParseResponse_ComplexFormatting(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantAnswer string
		wantRationale string
	}{
		{
			name: "nested lists and sub-bullets",
			content: `# ANSWER

1. First step
- Sub-item A
- Sub-item B

2. Second step
- Another sub-item

# RATIONALE

Complex structure.`,
			wantAnswer: "1. First step\n- Sub-item A\n- Sub-item B\n\n2. Second step\n- Another sub-item",
			wantRationale: "Complex structure.",
		},
		{
			name: "inline code in answer",
			content: `# ANSWER

Use ` + "`activated charcoal`" + ` immediately.

# RATIONALE

Test.`,
			wantAnswer: "Use `activated charcoal` immediately.",
			wantRationale: "Test.",
		},
		{
			name: "multiline with indentation",
			content: `# ANSWER

Main point:
  - Indented item
  - Another indented

# RATIONALE

Test.`,
			wantAnswer: "Main point:\n  - Indented item\n  - Another indented",
			wantRationale: "Test.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reply := ParseResponse(tt.content)
			if reply.Answer != tt.wantAnswer {
				t.Errorf("Expected answer %q, got %q", tt.wantAnswer, reply.Answer)
			}
			if reply.Rationale != tt.wantRationale {
				t.Errorf("Expected rationale %q, got %q", tt.wantRationale, reply.Rationale)
			}
		})
	}
}

// TestParseResponse_Discussion verifies parsing of discussion threads with multiple agents
func TestParseResponse_Discussion(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantDiscussion map[string]string
	}{
		{
			name: "single discussion entry",
			content: `# ANSWER

Test answer

# DISCUSSION

## With GPT

Your analysis is thorough.`,
			wantDiscussion: map[string]string{
				"GPT": "Your analysis is thorough.",
			},
		},
		{
			name: "multiple discussion entries",
			content: `# ANSWER

Test answer

# DISCUSSION

## With GPT

Consider adding more data.

## With Claude

Good approach overall.`,
			wantDiscussion: map[string]string{
				"GPT": "Consider adding more data.",
				"Claude": "Good approach overall.",
			},
		},
		{
			name: "discussion with multiline messages",
			content: `# ANSWER

Test answer

# DISCUSSION

## With GPT

First line.
Second line.
Third line.`,
			wantDiscussion: map[string]string{
				"GPT": "First line.\nSecond line.\nThird line.",
			},
		},
		{
			name: "no discussion section",
			content: `# ANSWER

Test answer

# RATIONALE

Test.`,
			wantDiscussion: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reply := ParseResponse(tt.content)
			if len(reply.Discussion) != len(tt.wantDiscussion) {
				t.Errorf("Expected %d discussion entries, got %d", len(tt.wantDiscussion), len(reply.Discussion))
			}
			for agent, expected := range tt.wantDiscussion {
				if reply.Discussion[agent] != expected {
					t.Errorf("For agent %s, expected %q, got %q", agent, expected, reply.Discussion[agent])
				}
			}
		})
	}
}

// TestParseResponse_RawContentPreserved verifies that original content is stored in RawContent field
func TestParseResponse_RawContentPreserved(t *testing.T) {
	content := `# ANSWER

Test answer

# RATIONALE

Test rationale`

	reply := ParseResponse(content)
	
	if reply.RawContent != content {
		t.Error("RawContent should be preserved exactly")
	}
}

// TestParseResponse_EdgeCases verifies handling of empty input, whitespace, and malformed headings
func TestParseResponse_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		content string
		check   func(*testing.T, types.Reply)
	}{
		{
			name:    "empty string",
			content: "",
			check: func(t *testing.T, r types.Reply) {
				if r.Answer != "" || r.Rationale != "" || len(r.Discussion) != 0 {
					t.Error("Empty input should produce empty reply")
				}
			},
		},
		{
			name:    "only whitespace",
			content: "   \n\n   \n",
			check: func(t *testing.T, r types.Reply) {
				if r.Answer != "" {
					t.Error("Whitespace-only input should produce empty answer")
				}
			},
		},
		{
			name: "headings without content",
			content: `# ANSWER
# RATIONALE
# DISCUSSION`,
			check: func(t *testing.T, r types.Reply) {
				if r.Answer != "" || r.Rationale != "" {
					t.Error("Headings without content should produce empty fields")
				}
			},
		},
		{
			name: "case sensitivity of headings",
			content: `# answer

Test

# RATIONALE

Test`,
			check: func(t *testing.T, r types.Reply) {
				// Lowercase 'answer' should not be recognized
				if r.Answer != "" {
					t.Error("Lowercase heading should not be recognized")
				}
			},
		},
		{
			name: "extra whitespace in headings",
			content: `#  ANSWER  

Test

# RATIONALE

Test`,
			check: func(t *testing.T, r types.Reply) {
				if r.Answer != "Test" {
					t.Errorf("Extra whitespace in heading should be handled, got %q", r.Answer)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reply := ParseResponse(tt.content)
			tt.check(t, reply)
		})
	}
}
