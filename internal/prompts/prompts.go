package prompts

import "fmt"

const ProcessDesc = "You are an AI assistant tasked with answering a question through iterative refinement. For each round, provide a refined answer and suggestions for further improvement."

func InitialPrompt(question string) string {
	return fmt.Sprintf("Question: %s\n\n%s\n\nProvide a refined answer and suggestions.", question, ProcessDesc)
}

func RefinePrompt(question, context string) string {
	return fmt.Sprintf("Question: %s\n\nPrevious context:\n%s\n\n%s\n\nRefine the answer further and provide new suggestions.", question, context, ProcessDesc)
}

func RankPrompt(question, context string, options []string) string {
	prompt := fmt.Sprintf("Question: %s\n\nContext:\n%s\n\nRank the following options from best to worst (A > B > C means A is best):\n", question, context)
	for i, opt := range options {
		prompt += fmt.Sprintf("%d. %s\n", i+1, opt)
	}
	return prompt + "\nProvide the ranking in the format: A > B > C"
}
