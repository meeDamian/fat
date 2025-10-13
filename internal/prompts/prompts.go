package prompts

import (
	"fmt"

	"github.com/meedamian/fat/internal/types"
)

const ProcessDesc = "You are an AI assistant tasked with answering a question through iterative refinement. For each round, provide a refined answer and suggestions for further improvement."

func InitialPrompt(question string) string {
	return question
}

func RefinePrompt(question, context string, mi *types.ModelInfo, round, numRounds int, activeModels []*types.ModelInfo) string {
	prompt := fmt.Sprintf(`You are agent %s in a %d-agent collaboration on the original question: "%s". This is round %d of %d. Review all prior agent outputs from round %d:

%s

1. Critically analyze: Identify gaps in your prior answer (e.g., missed facts, alternative POVs, biases). Challenge assumptions—debate why others might be wrong/right. Refine your answer: Improve for accuracy/depth, or repeat verbatim if no value-add.

2. Incorporate targeted suggestions to you (e.g., from Agent Z: "Consider [aspect]").

3. Generate 1-3 concise, actionable suggestions for other agents (only if they missed key angles; format as "To Agent [NAME]: Bullet on [specific improvement, e.g., 'Incorporate data from X to address Y bias']").

Output strict format—no extra text:

YOUR REFINED ANSWER

--- SUGGESTIONS ---

To Agent X: Suggestion bullet.
To Agent Z: Another.`, mi.Name, len(activeModels), question, round+1, numRounds, round, context)
	return prompt
}

func FinalPrompt(question, context string) string {
	return fmt.Sprintf("Question: %s\n\nPrevious context:\n%s\n\nThis is the final round. After considering everything from previous rounds, reply only with your final answer.", question, context)
}

func RankPrompt(question, context string, activeModels []*types.ModelInfo) string {
	prompt := fmt.Sprintf("Question: %s\n\nContext:\n%s\n\nRank the models from best to worst:\n", question, context)
	for _, mi := range activeModels {
		prompt += fmt.Sprintf("- %s\n", mi.Name)
	}
	return prompt + "\nPick the best model from the list above. Respond with only the model name."
}
