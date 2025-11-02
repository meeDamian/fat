package constants

import (
	_ "embed"
	"strings"
)

//go:embed questions.txt
var questionsFile string

// SampleQuestions contains a curated list of interesting prompts for the AI models
// Loaded from questions.txt (one question per line)
var SampleQuestions []string

func init() {
	// Parse questions from embedded file
	lines := strings.SplitSeq(questionsFile, "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			SampleQuestions = append(SampleQuestions, line)
		}
	}
}
