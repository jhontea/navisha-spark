package content

import (
	"fmt"
	"strings"
)

// PromptType defines the type of prompt to generate.
type PromptType string

const (
	PromptTypeInsight   PromptType = "insight"
	PromptTypeFollowUp  PromptType = "follow_up"
	PromptTypeVariation PromptType = "variation"
)

// PromptConfig holds configuration for prompt generation.
type PromptConfig struct {
	Model       string
	MaxTokens   int
	Temperature float64
}

// PromptBuilder builds prompts for LLM content generation.
type PromptBuilder struct {
	config PromptConfig
}

// NewPromptBuilder creates a new PromptBuilder.
func NewPromptBuilder(cfg PromptConfig) *PromptBuilder {
	return &PromptBuilder{config: cfg}
}

// BuildInsightPrompt builds a prompt for generating a new insight.
func (pb *PromptBuilder) BuildInsightPrompt(category, level, subtopic string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf(`Generate a %s level learning insight about %s topic: %s.

Requirements:
1. Title: Clear and specific (max 100 chars)
2. Insight: Comprehensive explanation (300-500 words) with:
   - Definition and core concepts
   - Practical examples and use cases
   - Code snippets if applicable
   - When to use / when not to use
   - Common pitfalls and best practices
3. Key Points: 3-5 bullet points summarizing the insight
4. Follow-ups: 2-3 questions with detailed answers (each answer 100-200 words)
5. Tags: 3-5 relevant tags

Respond ONLY with valid JSON in this exact format (no markdown, no code blocks):
{
    "title": "...",
    "insight": "...",
    "key_points": ["..."],
    "code_example": "...",
    "follow_ups": [
        {"q": "...", "a": "..."}
    ],
    "tags": ["..."]
}`, level, category, subtopic))

	return b.String()
}

// BuildVariationPrompt builds a prompt for creating a variation of an existing insight.
func (pb *PromptBuilder) BuildVariationPrompt(category, level string, existingInsight string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf(`Create a variation of the following %s level insight about %s.
The variation should cover a related but different aspect or angle of the same topic.

Original insight:
%s

Requirements:
1. Title: Different from the original (max 100 chars)
2. Insight: Comprehensive explanation (300-500 words) covering a different aspect
3. Key Points: 3-5 bullet points
4. Follow-ups: 2-3 questions with answers
5. Tags: 3-5 relevant tags

Respond ONLY with valid JSON in this exact format:
{
    "title": "...",
    "insight": "...",
    "key_points": ["..."],
    "code_example": "...",
    "follow_ups": [
        {"q": "...", "a": "..."}
    ],
    "tags": ["..."]
}`, level, category, existingInsight))

	return b.String()
}

// BuildFollowUpPrompt builds a prompt for generating follow-up questions.
func (pb *PromptBuilder) BuildFollowUpPrompt(category, level, title, insight string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf(`Generate 2-3 follow-up questions with detailed answers for the following %s level insight about %s.

Title: %s
Insight: %s

Requirements:
- Each question should test deeper understanding
- Each answer should be 100-200 words
- Cover practical implications and edge cases

Respond ONLY with valid JSON in this exact format:
{
    "follow_ups": [
        {"q": "...", "a": "..."},
        {"q": "...", "a": "..."}
    ]
}`, level, category, title, insight))

	return b.String()
}

// BuildSystemPrompt builds the system prompt for the LLM.
func (pb *PromptBuilder) BuildSystemPrompt() string {
	return `You are a senior backend engineering tutor creating high-quality learning content. 
Your audience is experienced backend engineers who want to deepen their knowledge.

Guidelines:
1. Be technically accurate and precise
2. Include practical, real-world examples
3. Explain trade-offs and when to use different approaches
4. Focus on concepts that matter for senior-level understanding
5. Use clear, professional language
6. Include code examples in Go when relevant
7. Always provide code examples in proper syntax without markdown formatting

CRITICAL: Your response must be ONLY valid JSON. No markdown formatting, no code blocks, no explanation.`
}

// GeneratePrompt generates a complete prompt with system message and user message.
func (pb *PromptBuilder) GeneratePrompt(promptType PromptType, params map[string]string) string {
	switch promptType {
	case PromptTypeInsight:
		return pb.BuildInsightPrompt(
			params["category"],
			params["level"],
			params["subtopic"],
		)
	case PromptTypeVariation:
		return pb.BuildVariationPrompt(
			params["category"],
			params["level"],
			params["existing_insight"],
		)
	case PromptTypeFollowUp:
		return pb.BuildFollowUpPrompt(
			params["category"],
			params["level"],
			params["title"],
			params["insight"],
		)
	default:
		return ""
	}
}
