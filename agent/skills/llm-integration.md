# LLM Integration (OpenRouter) — Skill Guide

## Overview

Skill ini berisi panduan lengkap untuk integrasi LLM (OpenRouter) dalam Navisha Spark. Mencakup API integration, prompt engineering, response parsing, error handling, dan best practices untuk generate questions on-the-fly.

---

## 1. OpenRouter Setup

### 1.1 Get API Key

1. Buka [OpenRouter.ai](https://openrouter.ai/)
2. Sign up / Login
3. Buka **Keys** section
4. Create new API key
5. Simpan API key (format: `sk-or-v1-...`)

**Navisha Spark API Key:**
```
sk-or-v1-your-api-key-here
```

### 1.2 Available Models

**Free Models (Recommended untuk Navisha Spark):**
- `openrouter/owl-alpha` — Default model
- `mistralai/mistral-7b-instruct` — Mistral 7B
- `meta-llama/llama-3-8b-instruct` — Llama 3 8B
- `microsoft/phi-3-mini-128k-instruct` — Phi-3 Mini

**Paid Models (Optional, untuk better quality):**
- `openai/gpt-4o` — GPT-4o
- `anthropic/claude-3-5-sonnet` — Claude 3.5 Sonnet
- `google/gemini-pro-1.5` — Gemini Pro

### 1.3 Model Configuration

```go
type LLMConfig struct {
    Model       string
    MaxTokens   int
    Temperature float64
    TopP        float64
}

var DefaultLLMConfig = LLMConfig{
    Model:       "openrouter/owl-alpha",
    MaxTokens:   1000,
    Temperature: 0.7,
    TopP:        0.9,
}
```

---

## 2. API Integration

### 2.1 Basic Client Setup

```go
import (
    "context"
    "fmt"
    "logrus"
    "time"
    
    "github.com/openai/openai-go"
    "github.com/openai/openai-go/option"
)

type LLMClient struct {
    client *openai.Client
    config LLMConfig
    logger *logrus.Entry
}

func NewLLMClient(apiKey string, config LLMConfig, logger *logrus.Entry) *LLMClient {
    client := openai.NewClient(
        option.WithAPIKey(apiKey),
        option.WithBaseURL("https://openrouter.ai/api/v1"),
    )

    return &LLMClient{
        client: client,
        config: config,
        logger: logger,
    }
}
```

### 2.2 Simple Completion

```go
func (c *LLMClient) Complete(ctx context.Context, prompt string) (string, error) {
    c.logger.WithField("prompt_length", len(prompt)).Debug("Sending completion request")

    resp, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
        Model: openai.String(c.config.Model),
        Messages: []openai.ChatCompletionMessageParamUnion{
            openai.UserMessage(prompt),
        },
        MaxTokens:   openai.Int(c.config.MaxTokens),
        Temperature: openai.Float(c.config.Temperature),
        TopP:        openai.Float(c.config.TopP),
    })
    if err != nil {
        return "", fmt.Errorf("failed to get completion: %w", err)
    }

    if len(resp.Choices) == 0 {
        return "", fmt.Errorf("no completion choices returned")
    }

    content := resp.Choices[0].Message.Content
    c.logger.WithFields(logrus.Fields{
        "response_length": len(content),
        "model":           c.config.Model,
    }).Debug("Completion received")

    return content, nil
}
```

### 2.3 Streaming Completion (Optional)

```go
func (c *LLMClient) CompleteStream(ctx context.Context, prompt string) (<-chan string, error) {
    stream := c.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
        Model: openai.String(c.config.Model),
        Messages: []openai.ChatCompletionMessageParamUnion{
            openai.UserMessage(prompt),
        },
        MaxTokens:   openai.Int(c.config.MaxTokens),
        Temperature: openai.Float(c.config.Temperature),
    })

    resultChan := make(chan string)
    go func() {
        defer close(resultChan)
        
        for stream.Next() {
            chunk := stream.Current()
            if len(chunk.Choices) > 0 {
                resultChan <- chunk.Choices[0].Delta.Content
            }
        }
        
        if err := stream.Err(); err != nil {
            c.logger.WithError(err).Error("Streaming error")
        }
    }()

    return resultChan, nil
}
```

---

## 3. Question Generation

### 3.1 Prompt Template

```go
const QuestionGenerationPrompt = `You are a backend engineering expert. Generate a {level} level question about {category} topic: {subtopic}.

Requirements:
1. Question must be technically accurate and specific
2. Answer must be detailed, correct, and comprehensive (200-500 words)
3. Include 2-3 follow-up questions that test deeper understanding
4. Focus on practical, real-world scenarios
5. Use proper technical terminology

Format your response as JSON:
{
    "question": "The question text",
    "answer": "The detailed answer",
    "follow_ups": ["Follow-up 1", "Follow-up 2", "Follow-up 3"],
    "tags": ["tag1", "tag2", "tag3"]
}

Level guidelines:
- beginner: Basic concepts, definitions, simple use cases
- intermediate: Practical implementation, trade-offs, best practices
- advanced: Deep internals, edge cases, system design implications

Category: {category}
Level: {level}
Subtopic: {subtopic}

Generate the question now:`
```

### 3.2 Generate Question Function

```go
type GeneratedQuestion struct {
    Question  string   `json:"question"`
    Answer    string   `json:"answer"`
    FollowUps []string `json:"follow_ups"`
    Tags      []string `json:"tags"`
}

func (c *LLMClient) GenerateQuestion(
    ctx context.Context,
    category, level, subtopic string,
) (*GeneratedQuestion, error) {
    prompt := fmt.Sprintf(QuestionGenerationPrompt,
        level, category, subtopic,
    )

    c.logger.WithFields(logrus.Fields{
        "category": category,
        "level":    level,
        "subtopic": subtopic,
    }).Info("Generating question via LLM")

    response, err := c.Complete(ctx, prompt)
    if err != nil {
        return nil, fmt.Errorf("failed to generate question: %w", err)
    }

    // Parse JSON response
    var gq GeneratedQuestion
    if err := json.Unmarshal([]byte(response), &gq); err != nil {
        c.logger.WithError(err).WithField("response", response).Error("Failed to parse LLM response")
        return nil, fmt.Errorf("invalid JSON response from LLM: %w", err)
    }

    // Validate generated question
    if err := c.validateGeneratedQuestion(&gq); err != nil {
        c.logger.WithError(err).Warn("Generated question failed validation")
        return nil, err
    }

    c.logger.WithFields(logrus.Fields{
        "question_length": len(gq.Question),
        "answer_length":   len(gq.Answer),
        "follow_ups":      len(gq.FollowUps),
    }).Info("Question generated successfully")

    return &gq, nil
}
```

### 3.3 Validation

```go
func (c *LLMClient) validateGeneratedQuestion(gq *GeneratedQuestion) error {
    if gq.Question == "" {
        return fmt.Errorf("question is empty")
    }
    if gq.Answer == "" {
        return fmt.Errorf("answer is empty")
    }
    if len(gq.Question) < 10 {
        return fmt.Errorf("question too short: %d chars", len(gq.Question))
    }
    if len(gq.Answer) < 50 {
        return fmt.Errorf("answer too short: %d chars", len(gq.Answer))
    }
    if len(gq.FollowUps) < 2 {
        return fmt.Errorf("not enough follow-up questions: %d", len(gq.FollowUps))
    }
    return nil
}
```

---

## 4. Follow-up Generation

### 4.1 Generate Follow-up Questions

```go
const FollowUpPrompt = `Given this backend engineering question and answer:

Question: %s
Answer: %s

Generate 2-3 follow-up questions that:
1. Test deeper understanding of the concept
2. Explore edge cases or trade-offs
3. Connect to related topics

Format as JSON array:
["follow-up 1", "follow-up 2", "follow-up 3"]`

func (c *LLMClient) GenerateFollowUps(ctx context.Context, question, answer string) ([]string, error) {
    prompt := fmt.Sprintf(FollowUpPrompt, question, answer)

    response, err := c.Complete(ctx, prompt)
    if err != nil {
        return nil, err
    }

    var followUps []string
    if err := json.Unmarshal([]byte(response), &followUps); err != nil {
        return nil, fmt.Errorf("invalid JSON response: %w", err)
    }

    return followUps, nil
}
```

### 4.2 Generate Answer Explanation

```go
const ExplanationPrompt = `Explain this backend engineering concept in detail:

Topic: %s
Question: %s

Provide a comprehensive explanation that:
1. Starts with a clear, concise definition
2. Explains how it works internally
3. Gives practical examples
4. Discusses pros and cons
5. Mentions common pitfalls

Keep the explanation between 200-500 words.`

func (c *LLMClient) GenerateExplanation(ctx context.Context, topic, question string) (string, error) {
    prompt := fmt.Sprintf(ExplanationPrompt, topic, question)
    return c.Complete(ctx, prompt)
}
```

---

## 5. Prompt Engineering

### 5.1 System Prompt

```go
var SystemPrompt = `You are an expert backend engineering instructor with deep knowledge in:
- Golang and concurrency patterns
- Data structures and algorithms
- System design and distributed systems
- Databases (SQL and NoSQL)
- API design and microservices
- DevOps and deployment
- Security best practices
- Caching and message brokers
- AI/ML for backend engineers

Your responses must be:
- Technically accurate and precise
- Well-structured and easy to understand
- Practical with real-world examples
- Free from hallucinations or guesswork

If you're unsure about something, say so explicitly.`
```

### 5.2 Few-Shot Examples

```go
const FewShotPrompt = `Generate backend engineering questions in the following format:

Example 1:
{
    "category": "Golang",
    "level": "beginner",
    "question": "Apa itu goroutine dan bagaimana cara membuatnya?",
    "answer": "Goroutine adalah thread ringan yang dikelola oleh Go runtime...",
    "follow_ups": ["Bagaimana cara menghentikan goroutine?", "Apa perbedaan goroutine dan thread OS?"],
    "tags": ["golang", "concurrency", "goroutine"]
}

Example 2:
{
    "category": "Database",
    "level": "intermediate",
    "question": "Jelaskan perbedaan antara indexing di PostgreSQL!",
    "answer": "Indexing di PostgreSQL menggunakan B-tree secara default...",
    "follow_ups": ["Kapan sebaiknya tidak menggunakan index?", "Apa itu covering index?"],
    "tags": ["database", "postgresql", "indexing"]
}

Now generate a new question following the same format:`
```

### 5.3 Chain-of-Thought Prompting

```go
const ChainOfThoughtPrompt = `Let's think step by step to generate a high-quality backend engineering question.

Step 1: Choose a specific subtopic from {category}
Step 2: Determine the difficulty level ({level})
Step 3: Think about what makes this topic important for senior backend engineers
Step 4: Formulate a question that tests practical understanding
Step 5: Write a comprehensive answer with examples
Step 6: Create follow-up questions that explore deeper

Topic: {category} - {subtopic}
Level: {level}

Generate the question:`
```

---

## 6. Response Parsing

### 6.1 JSON Parsing with Fallback

```go
func (c *LLMClient) ParseJSONResponse(response string, v interface{}) error {
    // Try direct parse
    if err := json.Unmarshal([]byte(response), v); err == nil {
        return nil
    }

    // Try to extract JSON from markdown code block
    re := regexp.MustCompile("```json\\s*([\\s\\S]*?)\\s*```")
    matches := re.FindStringSubmatch(response)
    if len(matches) > 1 {
        if err := json.Unmarshal([]byte(matches[1]), v); err == nil {
            return nil
        }
    }

    // Try to extract JSON from code block without language
    re = regexp.MustCompile("```\\s*([\\s\\S]*?)\\s*```")
    matches = re.FindStringSubmatch(response)
    if len(matches) > 1 {
        if err := json.Unmarshal([]byte(matches[1]), v); err == nil {
            return nil
        }
    }

    return fmt.Errorf("failed to parse JSON from response: %s", response)
}
```

### 6.2 Robust Parsing

```go
func (c *LLMClient) ParseGeneratedQuestion(response string) (*GeneratedQuestion, error) {
    // Clean response
    response = strings.TrimSpace(response)
    
    // Remove markdown code blocks if present
    response = strings.ReplaceAll(response, "```json", "")
    response = strings.ReplaceAll(response, "```", "")
    response = strings.TrimSpace(response)

    var gq GeneratedQuestion
    if err := json.Unmarshal([]byte(response), &gq); err != nil {
        return nil, fmt.Errorf("failed to parse question: %w", err)
    }

    return &gq, nil
}
```

---

## 7. Error Handling

### 7.1 Common Errors

```go
func (c *LLMClient) HandleLLMError(err error) error {
    if err == nil {
        return nil
    }

    // Check for OpenAI/OpenRouter specific errors
    var reqErr *openai.RequestError
    if errors.As(err, &reqErr) {
        switch reqErr.HTTPStatusCode {
        case 400:
            return fmt.Errorf("bad request (invalid prompt): %w", err)
        case 401:
            return fmt.Errorf("unauthorized (invalid API key): %w", err)
        case 403:
            return fmt.Errorf("forbidden (rate limit or no access): %w", err)
        case 404:
            return fmt.Errorf("model not found: %w", err)
        case 429:
            return fmt.Errorf("rate limit exceeded: %w", err)
        case 500:
            return fmt.Errorf("OpenRouter server error: %w", err)
        default:
            return fmt.Errorf("OpenRouter API error %d: %w", reqErr.HTTPStatusCode, err)
        }
    }

    return err
}
```

### 7.2 Retry Logic

```go
func (c *LLMClient) GenerateWithRetry(
    ctx context.Context,
    category, level, subtopic string,
    maxRetries int,
) (*GeneratedQuestion, error) {
    var lastErr error
    for i := 0; i < maxRetries; i++ {
        q, err := c.GenerateQuestion(ctx, category, level, subtopic)
        if err == nil {
            return q, nil
        }

        lastErr = err
        c.logger.WithFields(logrus.Fields{
            "attempt": i + 1,
            "error":   err,
        }).Warn("LLM generation failed, retrying")

        if i < maxRetries-1 {
            // Exponential backoff: 2s, 4s, 8s
            delay := time.Duration(math.Pow(2, float64(i))) * time.Second
            time.Sleep(delay)
        }
    }

    return nil, fmt.Errorf("max retries (%d) reached: %w", maxRetries, lastErr)
}
```

### 7.3 Timeout Handling

```go
func (c *LLMClient) GenerateWithTimeout(
    ctx context.Context,
    category, level, subtopic string,
    timeout time.Duration,
) (*GeneratedQuestion, error) {
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    resultChan := make(chan *GeneratedQuestion)
    errorChan := make(chan error)

    go func() {
        q, err := c.GenerateQuestion(ctx, category, level, subtopic)
        if err != nil {
            errorChan <- err
            return
        }
        resultChan <- q
    }()

    select {
    case q := <-resultChan:
        return q, nil
    case err := <-errorChan:
        return nil, err
    case <-ctx.Done():
        return nil, ctx.Err()
    }
}
```

---

## 8. Cost Optimization

### 8.1 Token Counting

```go
func CountTokens(text string) int {
    // Rough estimation: 1 token ≈ 4 characters
    // For accurate counting, use tiktoken library
    return len(text) / 4
}

func (c *LLMClient) EstimateCost(prompt string) float64 {
    inputTokens := CountTokens(prompt)
    outputTokens := c.config.MaxTokens

    // OpenRouter pricing (example for openrouter/owl-alpha)
    // Check https://openrouter.ai/models for current pricing
    inputCostPer1K := 0.001  // $0.001 per 1K input tokens
    outputCostPer1K := 0.002 // $0.002 per 1K output tokens

    cost := (float64(inputTokens) / 1000.0 * inputCostPer1K) +
            (float64(outputTokens) / 1000.0 * outputCostPer1K)

    return cost
}
```

### 8.2 Caching

```go
type LLMCache struct {
    cache map[string]*CachedResponse
    mu    sync.RWMutex
    ttl   time.Duration
}

type CachedResponse struct {
    Response   string
    Timestamp  time.Time
}

func (c *LLMCache) Get(key string) (string, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    cached, exists := c.cache[key]
    if !exists {
        return "", false
    }

    if time.Since(cached.Timestamp) > c.ttl {
        delete(c.cache, key)
        return "", false
    }

    return cached.Response, true
}

func (c *LLMCache) Set(key, response string) {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.cache[key] = &CachedResponse{
        Response:  response,
        Timestamp: time.Now(),
    }
}

// Usage
func (c *LLMClient) GenerateWithCache(ctx context.Context, category, level, subtopic string) (*GeneratedQuestion, error) {
    cacheKey := fmt.Sprintf("%s:%s:%s", category, level, subtopic)
    
    if cached, found := c.cache.Get(cacheKey); found {
        c.logger.WithField("cache_key", cacheKey).Info("Using cached response")
        return c.ParseGeneratedQuestion(cached)
    }

    q, err := c.GenerateQuestion(ctx, category, level, subtopic)
    if err != nil {
        return nil, err
    }

    // Cache the result
    response, _ := json.Marshal(q)
    c.cache.Set(cacheKey, string(response))

    return q, nil
}
```

---

## 9. Advanced Features

### 9.1 Multi-turn Conversation

```go
type Conversation struct {
    Messages []openai.ChatCompletionMessageParamUnion
    MaxTurns int
}

func (c *LLMClient) Chat(ctx context.Context, conv *Conversation, userMessage string) (string, error) {
    // Add user message
    conv.Messages = append(conv.Messages, openai.UserMessage(userMessage))

    // Limit conversation history
    if len(conv.Messages) > conv.MaxTurns*2 {
        conv.Messages = conv.Messages[len(conv.Messages)-conv.MaxTurns*2:]
    }

    resp, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
        Model:    openai.String(c.config.Model),
        Messages: conv.Messages,
    })
    if err != nil {
        return "", err
    }

    assistantMessage := resp.Choices[0].Message.Content
    conv.Messages = append(conv.Messages, openai.AssistantMessage(assistantMessage))

    return assistantMessage, nil
}
```

### 9.2 Function Calling (Structured Output)

```go
// Note: OpenRouter supports function calling for compatible models

func (c *LLMClient) GenerateQuestionStructured(ctx context.Context, category, level, subtopic string) (*GeneratedQuestion, error) {
    tools := []openai.ChatCompletionFunctionToolParam{
        {
            Type: "function",
            Function: openai.FunctionDefinitionParam{
                Name:        "generate_question",
                Description: "Generate a backend engineering question",
                Parameters: map[string]interface{}{
                    "type": "object",
                    "properties": map[string]interface{}{
                        "question": map[string]interface{}{
                            "type":        "string",
                            "description": "The question text",
                        },
                        "answer": map[string]interface{}{
                            "type":        "string",
                            "description": "The detailed answer",
                        },
                        "follow_ups": map[string]interface{}{
                            "type": "array",
                            "items": map[string]interface{}{
                                "type": "string",
                            },
                        },
                        "tags": map[string]interface{}{
                            "type": "array",
                            "items": map[string]interface{}{
                                "type": "string",
                            },
                        },
                    },
                    "required": []string{"question", "answer", "follow_ups", "tags"},
                },
            },
        },
    }

    resp, err := c.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
        Model:    openai.String(c.config.Model),
        Messages: []openai.ChatCompletionMessageParamUnion{openai.UserMessage(prompt)},
        Tools:    tools,
    })
    if err != nil {
        return nil, err
    }

    // Parse function call
    if len(resp.Choices) > 0 && resp.Choices[0].Message.ToolCalls != nil {
        for _, toolCall := range resp.Choices[0].Message.ToolCalls {
            if toolCall.Function.Name == "generate_question" {
                var gq GeneratedQuestion
                if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &gq); err != nil {
                    return nil, err
                }
                return &gq, nil
            }
        }
    }

    return nil, fmt.Errorf("no function call in response")
}
```

---

## 10. Testing

### 10.1 Unit Test with Mock

```go
import "github.com/stretchr/testify/mock"

type MockLLMClient struct {
    mock.Mock
}

func (m *MockLLMClient) Complete(ctx context.Context, prompt string) (string, error) {
    args := m.Called(ctx, prompt)
    return args.String(0), args.Error(1)
}

func TestGenerateQuestion(t *testing.T) {
    mockClient := new(MockLLMClient)
    mockClient.On("Complete", mock.Anything, mock.Anything).Return(`{
        "question": "Test question?",
        "answer": "Test answer",
        "follow_ups": ["Follow-up 1", "Follow-up 2"],
        "tags": ["test"]
    }`, nil)

    client := NewLLMClient("test-key", DefaultLLMConfig, logrus.NewEntry(logrus.StandardLogger()))
    // Replace client.client with mock
    
    q, err := client.GenerateQuestion(context.Background(), "Golang", "beginner", "goroutine")
    if err != nil {
        t.Errorf("GenerateQuestion() error = %v", err)
    }
    if q.Question != "Test question?" {
        t.Errorf("GenerateQuestion() = %v, want 'Test question?'", q.Question)
    }

    mockClient.AssertExpectations(t)
}
```

### 10.2 Integration Test

```go
func TestOpenRouterIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    apiKey := os.Getenv("OPENROUTER_API_KEY")
    if apiKey == "" {
        t.Fatal("OPENROUTER_API_KEY not set")
    }

    client := NewLLMClient(apiKey, DefaultLLMConfig, logrus.NewEntry(logrus.StandardLogger()))
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    q, err := client.GenerateQuestion(ctx, "Golang", "beginner", "goroutine")
    if err != nil {
        t.Fatalf("GenerateQuestion() error = %v", err)
    }

    if q.Question == "" {
        t.Error("Generated question is empty")
    }
    if q.Answer == "" {
        t.Error("Generated answer is empty")
    }
    if len(q.FollowUps) < 2 {
        t.Errorf("Not enough follow-ups: %d", len(q.FollowUps))
    }

    t.Logf("Generated question: %s", q.Question)
    t.Logf("Answer length: %d", len(q.Answer))
}
```

---

## 11. Best Practices

### 11.1 Do's

✅ **Always set temperature** (0.7 untuk balance creativity/accuracy)  
✅ **Use system prompts** untuk set context  
✅ **Validate LLM responses** before using  
✅ **Implement retry logic** untuk transient errors  
✅ **Cache frequent requests** untuk reduce cost  
✅ **Log token usage** untuk cost tracking  
✅ **Set max tokens** untuk control response length  
✅ **Use structured output** (JSON) untuk parsing  
✅ **Handle timeouts** properly  
✅ **Monitor API usage** via OpenRouter dashboard

### 11.2 Don'ts

❌ **Don't trust LLM output blindly** — always validate  
❌ **Don't use LLM for critical facts** (use curated questions)  
❌ **Don't set temperature too high** (>0.9) for technical content  
❌ **Don't ignore rate limits**  
❌ **Don't hardcode API keys**  
❌ **Don't send sensitive data** to LLM  
❌ **Don't use streaming** unless necessary (more complex)  
❌ **Don't forget to handle JSON parse errors**

---

## 12. Prompt Templates Library

### 12.1 Question Templates by Category

```go
var QuestionTemplates = map[string]string{
    "Golang": `Generate a %s level Golang question about %s.
Focus on: concurrency, goroutines, channels, GMP scheduler, or memory management.
Make it practical and relevant for senior backend engineers.`,

    "Database": `Generate a %s level Database question about %s.
Focus on: indexing, query optimization, transactions, or replication.
Include specific database (PostgreSQL) examples.`,

    "System Design": `Generate a %s level System Design question about %s.
Focus on: scalability, reliability, trade-offs, and real-world scenarios.
Make it suitable for technical interview preparation.`,

    "Security": `Generate a %s level Security question about %s.
Focus on: authentication, authorization, OWASP Top 10, or cryptography.
Include practical examples and mitigation strategies.`,
}
```

### 12.2 Difficulty Modifiers

```go
var DifficultyModifiers = map[string]string{
    "beginner": `
Difficulty: BEGINNER
- Focus on basic concepts and definitions
- Use simple, clear language
- Provide straightforward examples
- Suitable for someone learning the topic`,

    "intermediate": `
Difficulty: INTERMEDIATE
- Focus on practical implementation
- Include trade-offs and best practices
- Use real-world scenarios
- Suitable for practicing backend engineers`,

    "advanced": `
Difficulty: ADVANCED
- Focus on deep internals and edge cases
- Include system design implications
- Challenge assumptions
- Suitable for senior backend engineers preparing for interviews`,
}
```

---

## 13. Monitoring & Observability

### 13.1 Logging

```go
func (c *LLMClient) logRequest(ctx context.Context, prompt string) {
    c.logger.WithFields(logrus.Fields{
        "model":       c.config.Model,
        "prompt_len":  len(prompt),
        "max_tokens":  c.config.MaxTokens,
        "temperature": c.config.Temperature,
    }).Info("LLM request sent")
}

func (c *LLMClient) logResponse(ctx context.Context, response string, duration time.Duration) {
    c.logger.WithFields(logrus.Fields{
        "response_len": len(response),
        "duration_ms":  duration.Milliseconds(),
        "model":        c.config.Model,
    }).Info("LLM response received")
}
```

### 13.2 Metrics

```go
type LLMMetrics struct {
    TotalRequests   int
    TotalTokens     int
    TotalCost       float64
    AverageLatency  time.Duration
    ErrorCount      int
}

func (c *LLMClient) CollectMetrics() LLMMetrics {
    // Implementation
    return LLMMetrics{}
}
```

---

## 14. OpenRouter Specific

### 14.1 API Endpoint

```
Base URL: https://openrouter.ai/api/v1
Auth: Bearer sk-or-v1-...
```

### 14.2 Request Format

```go
type OpenRouterRequest struct {
    Model       string    `json:"model"`
    Messages    []Message `json:"messages"`
    MaxTokens   int       `json:"max_tokens,omitempty"`
    Temperature float64   `json:"temperature,omitempty"`
    TopP        float64   `json:"top_p,omitempty"`
}

type Message struct {
    Role    string `json:"role"`
    Content string `json:"content"`
}
```

### 14.3 Response Format

```go
type OpenRouterResponse struct {
    ID      string   `json:"id"`
    Object  string   `json:"object"`
    Created int64    `json:"created"`
    Model   string   `json:"model"`
    Choices []Choice `json:"choices"`
    Usage   Usage    `json:"usage"`
}

type Choice struct {
    Index        int     `json:"index"`
    Message      Message `json:"message"`
    FinishReason string  `json:"finish_reason"`
}

type Usage struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`
    TotalTokens      int `json:"total_tokens"`
}
```

---

## 15. Troubleshooting

### 15.1 Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| `401 Unauthorized` | Invalid API key | Check API key di `.env` |
| `429 Too Many Requests` | Rate limit | Implement backoff, reduce frequency |
| `404 Not Found` | Model not available | Check model name di OpenRouter |
| `400 Bad Request` | Invalid prompt | Check prompt format |
| Timeout | Network issue | Increase timeout, add retry |
| Invalid JSON | LLM response format | Use robust JSON parsing |

### 15.2 Debug Mode

```go
// Enable debug logging
c.logger.Logger.SetLevel(logrus.DebugLevel)

// Log full request/response
c.logger.WithFields(logrus.Fields{
    "request":  prompt,
    "response": response,
}).Debug("Full LLM interaction")
```

---

**Document End**