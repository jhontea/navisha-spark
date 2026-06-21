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

	b.WriteString(fmt.Sprintf(`Buatkan insight pembelajaran level %s tentang topik %s: %s dalam bahasa Indonesia.

Requirements:
1. Title: Jelas dan spesifik (maks 100 karakter)
2. Insight: Penjelasan komprehensif (300-500 kata) dengan:
   - Definisi dan konsep inti
   - Contoh praktis dan use case
   - Code snippets jika applicable
   - Kapan menggunakan dan kapan tidak menggunakan
   - Common pitfalls dan best practices
3. Key Points: 3-5 bullet points yang merangkum insight
4. Follow-ups: 2-3 pertanyaan dengan jawaban detail (masing-masing 100-200 kata)
5. Tags: 3-5 tag yang relevan

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

	b.WriteString(fmt.Sprintf(`Buatkan variasi dari insight %s level tentang %s berikut dalam bahasa Indonesia.
Variasi ini harus mencakup aspek atau sudut pandang yang berbeda tetapi terkait dengan topik yang sama.

Insight asli:
%s

Requirements:
1. Title: Berbeda dari asli (maks 100 karakter)
2. Insight: Penjelasan komprehensif (300-500 kata) yang mencakup aspek berbeda
3. Key Points: 3-5 bullet points
4. Follow-ups: 2-3 pertanyaan dengan jawaban
5. Tags: 3-5 tag yang relevan

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

// BuildInsightPromptWithKey builds a prompt for generating a new insight for a specific key/topic.
func (pb *PromptBuilder) BuildInsightPromptWithKey(category, level, key string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf(`Buatkan insight pembelajaran level %s tentang topik %s dengan fokus pada: %s dalam bahasa Indonesia.

Requirements:
1. Title: Jelas dan spesifik (maks 100 karakter)
2. Insight: Penjelasan komprehensif (300-500 kata) dengan:
   - Definisi dan konsep inti
   - Contoh praktis dan use case
   - Code snippets jika applicable
   - Kapan menggunakan dan kapan tidak menggunakan
   - Common pitfalls dan best practices
3. Key Points: 3-5 bullet points yang merangkum insight
4. Follow-ups: 2-3 pertanyaan dengan jawaban detail (masing-masing 100-200 kata)
5. Tags: 3-5 tag yang relevan

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
}`, level, category, key))

	return b.String()
}

// BuildVariationPromptWithKey builds a prompt for creating a variation of an existing insight for a specific key.
func (pb *PromptBuilder) BuildVariationPromptWithKey(category, level, key, existingInsight string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf(`Buatkan variasi dari insight %s level tentang %s dengan topik %s berikut dalam bahasa Indonesia.
Variasi ini harus mencakup aspek atau sudut pandang yang berbeda tetapi terkait dengan topik yang sama.

Insight asli:
%s

Requirements:
1. Title: Berbeda dari asli (maks 100 karakter)
2. Insight: Penjelasan komprehensif (300-500 kata) yang mencakup aspek berbeda
3. Key Points: 3-5 bullet points
4. Follow-ups: 2-3 pertanyaan dengan jawaban
5. Tags: 3-5 tag yang relevan

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
}`, level, category, key, existingInsight))

	return b.String()
}

// BuildFollowUpPrompt builds a prompt for generating follow-up questions.
func (pb *PromptBuilder) BuildFollowUpPrompt(category, level, title, insight string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf(`Buatkan 2-3 pertanyaan follow-up dengan jawaban detail untuk insight %s level tentang %s berikut dalam bahasa Indonesia.

Title: %s
Insight: %s

Requirements:
- Setiap pertanyaan harus menguji pemahaman yang lebih dalam
- Setiap jawaban harus 100-200 kata
- Sertakan implikasi praktis dan edge cases

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
	return `You are a senior backend engineering tutor creating high-quality learning content in Indonesian language.
Your audience is experienced backend engineers who want to deepen their knowledge.

Guidelines:
1. Be technically accurate and precise
2. Include practical, real-world examples
3. Explain trade-offs and when to use different approaches
4. Focus on concepts that matter for senior-level understanding
5. Use clear, professional Indonesian language
6. Include code examples in Go when relevant
7. Provide code examples in proper syntax (no markdown code blocks in the JSON values)

CRITICAL INSTRUCTIONS:
- Your response must be ONLY valid JSON
- Do NOT escape markdown characters (like *, _, [, ], etc.) in the JSON string values
- Write plain text in the JSON values - the system will handle markdown formatting later
- No markdown formatting, no code blocks, no explanation outside the JSON structure
- Example: Write "SET key value" NOT "\SET\ key\ value" in the JSON strings`
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
