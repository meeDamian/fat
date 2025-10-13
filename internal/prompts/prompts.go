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

func FinalPrompt(question, context string) string {
	return fmt.Sprintf("Question: %s\n\nPrevious context:\n%s\n\nThis is the final round. After considering everything from previous rounds, reply only with your final answer.", question, context)
}

func RankPrompt(question, context string, activeModels []*types.ModelInfo) string {
	prompt := fmt.Sprintf("Question: %s\n\nContext:\n%s\n\nRank the models from best to worst:\n", question, context)
	for _, mi := range activeModels {
		prompt += fmt.Sprintf("- %s\n", mi.Name)
	}
	return prompt + "\nProvide the ranking in the format: model1 > model2 > model3"
}
