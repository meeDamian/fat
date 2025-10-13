package prompts

import (
	"fmt"

	"github.com/meedamian/fat/internal/types"
)

const ProcessDesc = "You are an AI assistant tasked with answering a question through iterative refinement. For each round, provide a refined answer and suggestions for further improvement."

func InitialPrompt(question string) string {
	return question
}

func RefinePrompt(question, context string) string {
	return fmt.Sprintf("Question: %s\n\nPrevious context:\n%s\n\n%s\n\nRefine the answer further and provide new suggestions.", question, context, ProcessDesc)
}

func RankPrompt(question, context string, activeModels []*types.ModelInfo) string {
	modelLetters := map[string]string{"grok": "A", "gpt": "B", "claude": "C", "gemini": "D"}
	prompt := fmt.Sprintf("Question: %s\n\nContext:\n%s\n\nRank the models from best to worst (A > B > C means A is best):\n", question, context)
	for _, mi := range activeModels {
		letter := modelLetters[mi.ID]
		prompt += fmt.Sprintf("%s: %s\n", letter, mi.Name)
	}
	return prompt + "\nProvide the ranking in the format: A > B > C"
}
