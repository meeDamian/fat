package prompts

import (
	"fmt"
	"strings"

	"github.com/meedamian/fat/internal/types"
)

const ProcessDesc = "You are an AI assistant tasked with answering a question through iterative refinement. For each round, provide a refined answer and suggestions for further improvement."

func InitialPrompt(question string) string {
	return question
}

func RefinePrompt(question, context string, mi *types.ModelInfo, round, numRounds int, activeModels []*types.ModelInfo) string {
	// Get short names of other agents for examples
	var otherShortNames []string
	for _, m := range activeModels {
		if m.ID != mi.ID {
			short := strings.Split(m.ID, "-")[0]
			otherShortNames = append(otherShortNames, short)
		}
	}
	example1 := "grok"
	example2 := "gemini"
	if len(otherShortNames) > 0 {
		example1 = otherShortNames[0]
	}
	if len(otherShortNames) > 1 {
		example2 = otherShortNames[1]
	}
	prompt := fmt.Sprintf(`You are agent %s in a %d-agent collaboration on the original question: "%s". This is round %d of %d. Review all prior agent outputs from round %d:

%s

1. Critically analyze: Identify gaps in your prior answer (e.g., missed facts, alternative POVs, biases). Challenge assumptions—debate why others might be wrong/right. Refine your answer: Improve for accuracy/depth, or repeat verbatim if no value-add.

2. Incorporate targeted suggestions to you (e.g., from %s: "Consider [aspect]").

3. Generate 1-3 concise, actionable suggestions for other agents (only if they missed key angles; format as "To Agent [NAME]: Bullet on [specific improvement, e.g., 'Incorporate data from X to address Y bias']").

Output strict format—no extra text:

YOUR REFINED ANSWER

--- SUGGESTIONS ---

To %s: Suggestion bullet.
To %s: Another.`, mi.Name, len(activeModels), question, round+1, numRounds, round, context, example1, example1, example2)
	return prompt
}

func FinalPrompt(question, context string, mi *types.ModelInfo, round, numRounds int, activeModels []*types.ModelInfo) string {
	prompt := fmt.Sprintf(`You are agent %s in a %d-agent collaboration on the original question: "%s". This is round %d of %d. Review all prior agent outputs from round %d:

%s

1. Critically analyze: Identify gaps in your prior answer (e.g., missed facts, alternative POVs, biases). Challenge assumptions—debate why others might be wrong/right. Refine your final answer: Improve for accuracy/depth, or repeat verbatim if no value-add.

2. Incorporate targeted suggestions to you (e.g., from grok: "Consider [aspect]").

Output strict format—no extra text:

YOUR FINAL ANSWER`, mi.Name, len(activeModels), question, round+1, numRounds, round, context)
	return prompt
}

func RankPrompt(question, context string, activeModels []*types.ModelInfo, mi *types.ModelInfo) string {
	prompt := fmt.Sprintf(`You are agent %s in a %d-agent collaboration on the original question: "%s". This is the ranking phase. Review the final answers from all agents:

%s

Rank all agents from best to worst based on their final answers. Respond with only the agent names in order: Best > Next > Worst`, mi.Name, len(activeModels), question, context)
	return prompt
}
