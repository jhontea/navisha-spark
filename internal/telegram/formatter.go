package telegram

import (
	"fmt"
	"strings"

	"github.com/lib/pq"
)

// InsightData represents the data needed to format a Telegram message.
type InsightData struct {
	Category    string
	Level       string
	Title       string
	Insight     string
	KeyPoints   []string
	CodeExample string
	FollowUps   []FollowUp
	Tags        []string
}

// FollowUp represents a follow-up question and answer.
type FollowUp struct {
	Question string `json:"q"`
	Answer   string `json:"a"`
}

// Formatter handles formatting insights into Telegram messages.
type Formatter struct {
	config FormatConfig
}

// FormatConfig holds formatting configuration.
type FormatConfig struct {
	IncludeCategory  bool
	IncludeLevel     bool
	IncludeFollowUps bool
	IncludeTags      bool
	MarkdownEnabled  bool
}

// NewFormatter creates a new Formatter.
func NewFormatter(cfg FormatConfig) *Formatter {
	return &Formatter{config: cfg}
}

// Format formats an insight into a Telegram message string.
func (f *Formatter) Format(data *InsightData) string {
	var b strings.Builder

	// Header with category and level
	if f.config.IncludeCategory || f.config.IncludeLevel {
		b.WriteString("📚 ")
		if f.config.IncludeCategory {
			b.WriteString("*")
			b.WriteString(escapeMarkdown(data.Category))
			b.WriteString("*")
		}
		if f.config.IncludeLevel {
			if f.config.IncludeCategory {
				b.WriteString(" — ")
			}
			b.WriteString("*")
			b.WriteString(formatLevel(data.Level))
			b.WriteString("*")
		}
		b.WriteString("\n\n")
	}

	// Title
	b.WriteString("*")
	b.WriteString(escapeMarkdown(data.Title))
	b.WriteString("*")
	b.WriteString("\n\n")

	// Insight section
	b.WriteString("📝 *Insight:*\n")
	b.WriteString(formatParagraphs(data.Insight))
	b.WriteString("\n\n")

	// Key points
	if len(data.KeyPoints) > 0 {
		b.WriteString("💡 *Key Points:*\n")
		for _, point := range data.KeyPoints {
			b.WriteString("• ")
			b.WriteString(escapeMarkdown(point))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Code example
	if data.CodeExample != "" {
		b.WriteString("💻 *Code Example:*\n")
		b.WriteString("```go\n")
		b.WriteString(data.CodeExample)
		b.WriteString("\n```\n\n")
	}

	// Follow-up questions
	if f.config.IncludeFollowUps && len(data.FollowUps) > 0 {
		b.WriteString("🔍 *Deep Dive:*\n")
		for i, fu := range data.FollowUps {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString("*Q: ")
			b.WriteString(escapeMarkdown(fu.Question))
			b.WriteString("*\n")
			b.WriteString("A: ")
			b.WriteString(escapeMarkdown(fu.Answer))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Tags
	if f.config.IncludeTags && len(data.Tags) > 0 {
		b.WriteString("---\n")
		b.WriteString("_Tags: ")
		b.WriteString(strings.Join(data.Tags, ", "))
		b.WriteString("_\n")
	}

	return b.String()
}

// FormatSimple formats a simple text message without structured formatting.
func (f *Formatter) FormatSimple(text string) string {
	return escapeMarkdown(text)
}

// GetLevelEmoji returns the emoji for a given level.
func GetLevelEmoji(level string) string {
	switch strings.ToLower(level) {
	case "beginner":
		return "🟢"
	case "intermediate":
		return "🟡"
	case "advanced":
		return "🔴"
	default:
		return "⚪"
	}
}

// formatLevel formats a level string with emoji and proper casing.
func formatLevel(level string) string {
	level = strings.Title(strings.ToLower(level))
	emoji := GetLevelEmoji(level)
	return fmt.Sprintf("%s %s", emoji, level)
}

// formatParagraphs formats text with proper paragraph breaks for better readability.
// It ensures paragraphs are separated by single newlines and removes excessive whitespace.
func formatParagraphs(text string) string {
	// First escape markdown
	escaped := escapeMarkdown(text)

	// Split by double newlines to identify paragraphs
	paragraphs := strings.Split(escaped, "\n\n")

	// Clean up each paragraph and join with single newlines
	var cleaned []string
	for _, p := range paragraphs {
		// Trim whitespace from each paragraph
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			// Replace multiple consecutive newlines with single newline within paragraph
			cleanedPara := strings.ReplaceAll(trimmed, "\n\n", "\n")
			cleaned = append(cleaned, cleanedPara)
		}
	}

	// Join paragraphs with double newline for visual separation
	return strings.Join(cleaned, "\n\n")
}

// escapeMarkdown escapes Telegram Markdown special characters.
func escapeMarkdown(text string) string {
	// Characters to escape in Markdown: _ * [ ] ( ) ~ ` > # + - = | { } !
	replacer := strings.NewReplacer(
		"_", "\\_",
		"*", "\\*",
		"[", "\\[",
		"]", "\\]",
		"(", "\\(",
		")", "\\)",
		"~", "\\~",
		"`", "\\`",
		">", "\\>",
		"#", "\\#",
		"+", "\\+",
		"-", "\\-",
		"=", "\\=",
		"|", "\\|",
		"{", "\\{",
		"}", "\\}",
		"!", "\\!",
	)
	return replacer.Replace(text)
}

// FormatInsightFromDB formats a database insight into a formatted message.
func (f *Formatter) FormatInsightFromDB(insight *InsightFromDB) string {
	data := &InsightData{
		Category:  insight.Category,
		Level:     insight.Level,
		Title:     insight.Title,
		Insight:   insight.Insight,
		KeyPoints: insight.KeyPoints,
		Tags:      insight.Tags,
	}

	if insight.CodeExample != nil {
		data.CodeExample = *insight.CodeExample
	}

	// Parse follow_ups from JSON if available
	// For now, use the FollowUps field if already parsed
	data.FollowUps = insight.FollowUpsList

	return f.Format(data)
}

// InsightFromDB is a helper struct for formatting insights from the database.
type InsightFromDB struct {
	Category      string
	Level         string
	Title         string
	Insight       string
	KeyPoints     pq.StringArray
	CodeExample   *string
	FollowUpsList []FollowUp
	Tags          pq.StringArray
}
