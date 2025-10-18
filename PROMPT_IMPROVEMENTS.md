# Prompt Template Analysis & Improvement Suggestions

## Current Template Analysis

The current prompt template is well-structured and follows best practices for multi-agent collaboration. It clearly:
- Identifies the agent and collaboration context
- Provides the question and previous round data
- Specifies strict response format requirements
- Encourages discussion and refinement

## Suggested Improvements

### 1. **Clarify Agent Identity Consistency**

**Current**: Uses both "agent [NAME]" and "Agent [NAME]" inconsistently

**Suggested**: Standardize to always use the model's display name (e.g., "Grok", "GPT") without "agent" prefix in discussion sections for clarity.

**Rationale**: Makes it easier for models to parse and reference each other correctly.

### 2. **Add Explicit Discussion Format Example**

**Current**: 
```
## With [AGENT_NAME]
[Your 1-2 new messages only, e.g., "Consider environmental counterpoint to your tech focus."]
```

**Suggested**: Add more concrete examples showing proper vs improper discussion format:

```markdown
# DISCUSSION

## With GPT
Your economic analysis is strong, but consider adding environmental impact data from the 2023 UN report.

## With Claude  
I agree with your ethical framework. Could you expand on the legal precedents you mentioned?

(Do NOT include: "To GPT:", "Agent GPT:", or other prefixes - just the message content)
```

**Rationale**: Reduces parsing errors and ensures consistent discussion format.

### 3. **Strengthen Round-Specific Instructions**

**Current**: Same instructions for all rounds

**Suggested**: Differentiate instructions by round:

**Round 1**:
```
This is round 1 - provide your initial analysis. No discussion section needed yet.
```

**Round 2+**:
```
This is round {N} of {TOTAL} - refine your answer based on:
1. Gaps identified in other agents' answers
2. Discussion points directed at you
3. New perspectives you can contribute

Provide 1-2 discussion messages to agents whose answers could benefit from your expertise.
```

**Rationale**: Prevents confusion about when discussion is required and sets clearer expectations per round.

### 4. **Add Word Count Guidance for Discussion**

**Current**: "1-2 concise messages per relevant agent"

**Suggested**: 
```
# DISCUSSION (Optional - skip if no substantive feedback)

## With [AGENT_NAME]
[One concise message, 20-50 words, focusing on a specific gap or improvement]
```

**Rationale**: Prevents overly verbose or vague discussion messages.

### 5. **Clarify "Refine" vs "Rewrite"**

**Current**: "Refine # ANSWER using # REPLIES + # DISCUSSION"

**Suggested**:
```
Refine your ANSWER by:
- Incorporating valid points from other agents' replies
- Addressing feedback directed at you in DISCUSSION
- Maintaining your core perspective while filling identified gaps
- NOT simply copying or paraphrasing other agents' work
```

**Rationale**: Prevents models from just copying the "best" answer instead of genuine refinement.

### 6. **Add Explicit Ranking Criteria**

**Current**: Ranking prompt mentions "quality" but is vague

**Suggested**: Add to ranking prompt:
```
Rank based on:
1. Factual accuracy (40%)
2. Completeness - addresses all aspects of question (30%)
3. Clarity and coherence (20%)
4. Integration of discussion points (10%)

Be objective - you may rank yourself anywhere based on these criteria.
```

**Rationale**: Provides clear, weighted criteria for more consistent rankings.

### 7. **Handle Edge Cases Explicitly**

**Suggested additions**:

```markdown
IMPORTANT RULES:
- If you have no substantive feedback for any agent, omit the # DISCUSSION section entirely
- If another agent's answer is clearly superior, acknowledge it in RATIONALE and explain your refinements
- Discussion messages should be constructive, not just praise or criticism
- Each discussion message must suggest a specific improvement or ask a clarifying question
```

**Rationale**: Reduces low-quality filler content and encourages meaningful collaboration.

## Recommended Updated Template

```markdown
You are {AGENT_NAME} in a {AGENT_COUNT}-agent collaboration. Other agents: {OTHER_AGENTS}. Round {ROUND} of {TOTAL_ROUNDS}.

--- QUESTION ---

{QUESTION_TEXT}

--- CONTEXT FROM PREVIOUS ROUND ---

# REPLIES

{FORMATTED_REPLIES}

# DISCUSSION DIRECTED AT YOU

{MESSAGES_FOR_THIS_AGENT}

# DISCUSSION YOU INITIATED

{MESSAGES_FROM_THIS_AGENT}

--- YOUR TASK ---

{ROUND_SPECIFIC_INSTRUCTIONS}

Respond in this EXACT format:

# ANSWER

{refined_answer_incorporating_feedback_max_300_words}

# RATIONALE

{optional_brief_explanation_of_changes_made}

# DISCUSSION

{optional_only_if_substantive_feedback_exists}

## With {AgentName}

{one_specific_actionable_suggestion_20_to_50_words}

--- EXAMPLE ---

Good discussion: "Your economic analysis omits inflation data from Q4 2023. Adding this would strengthen the GDP impact argument."

Bad discussion: "Good point!" or "I disagree with your approach."
```

## Implementation Priority

1. **High Priority**: Clarify discussion format (reduce parsing errors)
2. **High Priority**: Add round-specific instructions (improve collaboration quality)
3. **Medium Priority**: Standardize agent naming (reduce confusion)
4. **Medium Priority**: Add ranking criteria (improve winner selection)
5. **Low Priority**: Word count guidance (nice-to-have)

## Testing Recommendations

After implementing improvements:
1. Run 10+ test questions with all models
2. Measure discussion quality (% of messages with actionable feedback)
3. Measure ranking consistency (variance in Borda scores)
4. Check for parsing errors in response extraction
5. Validate that refinements actually incorporate feedback (not just rewrites)
