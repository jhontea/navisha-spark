package content

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// LLMResponse represents the response from the LLM API.
type LLMResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

// LLMRequest represents a request to the LLM API.
type LLMRequest struct {
	Model       string       `json:"model"`
	Messages    []LLMMessage `json:"messages"`
	MaxTokens   int          `json:"max_tokens,omitempty"`
	Temperature float64      `json:"temperature,omitempty"`
	Stream      bool         `json:"stream,omitempty"`
}

// LLMMessage represents a message in the LLM request.
type LLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GeneratedInsight represents a generated insight from the LLM.
type GeneratedInsight struct {
	Title       string     `json:"title"`
	Insight     string     `json:"insight"`
	KeyPoints   []string   `json:"key_points"`
	CodeExample string     `json:"code_example"`
	FollowUps   []FollowUp `json:"follow_ups"`
	Tags        []string   `json:"tags"`
}

// FollowUp represents a follow-up question and answer.
type FollowUp struct {
	Question string `json:"q"`
	Answer   string `json:"a"`
}

// Generator handles LLM content generation via OpenRouter.
type Generator struct {
	apiKey        string
	apiURL        string
	config        PromptConfig
	httpClient    *http.Client
	log           *logrus.Entry
	promptBuilder *PromptBuilder
}

// NewGenerator creates a new LLM content generator.
func NewGenerator(apiKey string, cfg PromptConfig, log *logrus.Entry) *Generator {
	return &Generator{
		apiKey: apiKey,
		apiURL: "https://openrouter.ai/api/v1/chat/completions",
		config: cfg,
		httpClient: &http.Client{
			Timeout: 180 * time.Second,
		},
		log:           log,
		promptBuilder: NewPromptBuilder(cfg),
	}
}

// GenerateInsight generates a new insight for the given category, level, and subtopic.
func (g *Generator) GenerateInsight(ctx context.Context, category, level, subtopic string) (*GeneratedInsight, error) {
	prompt := g.promptBuilder.BuildInsightPrompt(category, level, subtopic)
	systemPrompt := g.promptBuilder.BuildSystemPrompt()

	resp, err := g.callLLM(ctx, systemPrompt, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate insight: %w", err)
	}

	insight, err := g.parseInsightResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse insight response: %w", err)
	}

	g.log.WithFields(logrus.Fields{
		"category": category,
		"level":    level,
		"subtopic": subtopic,
		"title":    insight.Title,
	}).Info("insight generated successfully")

	return insight, nil
}

// GenerateVariation generates a variation of an existing insight.
func (g *Generator) GenerateVariation(ctx context.Context, category, level, existingInsight string) (*GeneratedInsight, error) {
	prompt := g.promptBuilder.BuildVariationPrompt(category, level, existingInsight)
	systemPrompt := g.promptBuilder.BuildSystemPrompt()

	resp, err := g.callLLM(ctx, systemPrompt, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate variation: %w", err)
	}

	insight, err := g.parseInsightResponse(resp)
	if err != nil {
		return nil, fmt.Errorf("failed to parse variation response: %w", err)
	}

	return insight, nil
}

// GenerateFollowUps generates follow-up questions for an existing insight.
func (g *Generator) GenerateFollowUps(ctx context.Context, category, level, title, insight string) ([]FollowUp, error) {
	prompt := g.promptBuilder.BuildFollowUpPrompt(category, level, title, insight)
	systemPrompt := g.promptBuilder.BuildSystemPrompt()

	resp, err := g.callLLM(ctx, systemPrompt, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate follow-ups: %w", err)
	}

	var result struct {
		FollowUps []FollowUp `json:"follow_ups"`
	}

	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		return nil, fmt.Errorf("failed to parse follow-ups response: %w", err)
	}

	return result.FollowUps, nil
}

// callLLM makes the actual API call to OpenRouter.
func (g *Generator) callLLM(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	reqBody := LLMRequest{
		Model: g.config.Model,
		Messages: []LLMMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   g.config.MaxTokens,
		Temperature: g.config.Temperature,
		Stream:      false,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.apiURL, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.apiKey))
	req.Header.Set("HTTP-Referer", "https://github.com/navisha/spark")
	req.Header.Set("X-Title", "Navisha Spark")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var llmResp LLMResponse
	if err := json.Unmarshal(respBody, &llmResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if llmResp.Error != nil {
		return "", fmt.Errorf("LLM API error: %s (type: %s, code: %s)",
			llmResp.Error.Message, llmResp.Error.Type, llmResp.Error.Code)
	}

	if len(llmResp.Choices) == 0 {
		return "", fmt.Errorf("LLM returned no choices")
	}

	content := llmResp.Choices[0].Message.Content

	g.log.WithFields(logrus.Fields{
		"model":         llmResp.Model,
		"tokens":        llmResp.Usage.TotalTokens,
		"finish_reason": llmResp.Choices[0].FinishReason,
	}).Debug("LLM API call completed")

	return content, nil
}

// parseInsightResponse parses the LLM response into a GeneratedInsight.
func (g *Generator) parseInsightResponse(response string) (*GeneratedInsight, error) {
	// Try direct JSON parse first
	var insight GeneratedInsight
	if err := json.Unmarshal([]byte(response), &insight); err == nil {
		if insight.Title != "" && insight.Insight != "" {
			return &insight, nil
		}
	}

	// Try to extract JSON from markdown code blocks
	cleaned := extractJSONFromResponse(response)
	if cleaned != "" {
		if err := json.Unmarshal([]byte(cleaned), &insight); err == nil {
			if insight.Title != "" && insight.Insight != "" {
				return &insight, nil
			}
		}
	}

	return nil, fmt.Errorf("failed to parse insight from response: %s", truncateString(response, 200))
}

// extractJSONFromResponse attempts to extract JSON from a response that may contain markdown.
func extractJSONFromResponse(response string) string {
	// Try to find JSON between ```json and ``` markers
	start := indexOf(response, "```json")
	if start >= 0 {
		start += 7 // len("```json")
		end := indexOf(response[start:], "```")
		if end >= 0 {
			return response[start : start+end]
		}
	}

	// Try to find JSON between ``` and ``` markers
	start = indexOf(response, "```")
	if start >= 0 {
		start += 3
		end := indexOf(response[start:], "```")
		if end >= 0 {
			return response[start : start+end]
		}
	}

	// Try to find JSON between { and }
	start = indexOf(response, "{")
	if start >= 0 {
		depth := 0
		for i := start; i < len(response); i++ {
			if response[i] == '{' {
				depth++
			} else if response[i] == '}' {
				depth--
				if depth == 0 {
					return response[start : i+1]
				}
			}
		}
	}

	return ""
}

// indexOf returns the index of the first occurrence of substr in s, or -1.
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// truncateString truncates a string to the specified length.
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
