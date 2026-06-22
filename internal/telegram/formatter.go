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

// Format formats an insight into one or more Telegram messages.
// If the message exceeds Telegram's 4096 character limit, it will be split into multiple messages.
func (f *Formatter) Format(data *InsightData) []string {
	var b strings.Builder

	// Header with category and level
	if f.config.IncludeCategory || f.config.IncludeLevel {
		b.WriteString("📚 ")
		if f.config.IncludeCategory {
			b.WriteString("*")
			b.WriteString(safeEscapeMarkdown(data.Category))
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
	b.WriteString(safeEscapeMarkdown(data.Title))
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
			b.WriteString(safeEscapeMarkdown(point))
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
			b.WriteString(safeEscapeMarkdown(fu.Question))
			b.WriteString("*\n")
			b.WriteString("A: ")
			b.WriteString(safeEscapeMarkdown(fu.Answer))
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

	message := b.String()

	// Split if exceeds Telegram's 4096 character limit
	const maxLength = 4096
	if len(message) > maxLength {
		return f.splitMessage(message, maxLength)
	}

	return []string{message}
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
	lower := strings.ToLower(level)
	// Capitalize first letter without using deprecated strings.Title
	if len(lower) > 0 {
		lower = strings.ToUpper(lower[:1]) + lower[1:]
	}
	emoji := GetLevelEmoji(lower)
	return fmt.Sprintf("%s %s", emoji, lower)
}

// formatParagraphs formats text with proper paragraph breaks for better readability.
// It ensures paragraphs are separated by single newlines and removes excessive whitespace.
// This function handles markdown escaping internally to prevent double-escaping.
func formatParagraphs(text string) string {
	// Unescape any pre-escaped markdown to avoid double-escaping
	// This handles cases where LLM already escaped characters like \( or \*
	unescaped := unescapeMarkdown(text)

	// Split by double newlines to identify paragraphs
	paragraphs := strings.Split(unescaped, "\n\n")

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
	joined := strings.Join(cleaned, "\n\n")

	// Re-escape markdown for Telegram
	return escapeMarkdown(joined)
}

// unescapeMarkdown reverses markdown escaping to prevent double-escaping.
func unescapeMarkdown(text string) string {
	replacer := strings.NewReplacer(
		"\\_", "_",
		"\\*", "*",
		"\\[", "[",
		"\\]", "]",
		"\\(", "(",
		"\\)", ")",
		"\\~", "~",
		"\\`", "`",
		"\\>", ">",
		"\\#", "#",
		"\\+", "+",
		"\\-", "-",
		"\\=", "=",
		"\\|", "|",
		"\\{", "{",
		"\\}", "}",
		"\\!", "!",
	)
	return replacer.Replace(text)
}

// splitMessage splits a long message into multiple parts that fit within Telegram's character limit.
// It tries to split at paragraph boundaries to avoid cutting off mid-sentence.
// When splitting, Markdown formatting is stripped from all parts except the first one to avoid parsing errors.
func (f *Formatter) splitMessage(message string, maxLength int) []string {
	if len(message) <= maxLength {
		return []string{message}
	}

	var messages []string
	remaining := message

	for len(remaining) > maxLength {
		// Try to find a good split point (double newline = paragraph break)
		splitAt := strings.LastIndex(remaining[:maxLength], "\n\n")
		if splitAt <= 0 {
			// Fallback: try single newline
			splitAt = strings.LastIndex(remaining[:maxLength], "\n")
		}
		if splitAt <= 0 {
			// Last resort: hard split at maxLength
			splitAt = maxLength
		}

		// Add the part
		part := strings.TrimSpace(remaining[:splitAt])
		if part != "" {
			messages = append(messages, part)
		}

		// Move to next part
		remaining = strings.TrimSpace(remaining[splitAt:])
	}

	// Add the remaining part
	if remaining != "" {
		messages = append(messages, remaining)
	}

	// Add part indicators (1/2, 2/2, etc.)
	// Strip markdown from all parts to avoid parsing errors when split
	totalParts := len(messages)
	for i := range messages {
		// Strip markdown formatting for split messages to avoid entity parsing errors
		plainText := f.stripMarkdown(messages[i])
		messages[i] = fmt.Sprintf("(%d/%d) %s", i+1, totalParts, plainText)
	}

	return messages
}

// stripMarkdown removes markdown formatting characters to create plain text.
// This is used for split messages where markdown entities might be broken.
func (f *Formatter) stripMarkdown(text string) string {
	// Remove markdown bold/italic markers
	replacer := strings.NewReplacer(
		"*", "",
		"_", "",
		"`", "",
		"~", "",
	)
	return replacer.Replace(text)
}

// safeEscapeMarkdown safely escapes markdown by first unescaping any pre-escaped content,
// then escaping it properly. This prevents double-escaping issues from LLM output.
func safeEscapeMarkdown(text string) string {
	// First unescape any pre-escaped markdown from LLM
	unescaped := unescapeMarkdown(text)
	// Then escape properly for Telegram
	return escapeMarkdown(unescaped)
}

// escapeMarkdown escapes Telegram Markdown special characters.
// Note: Parentheses () do NOT need escaping in Telegram Markdown.
// Hyphens (-) are only escaped when at the start of a line to prevent list interpretation.
func escapeMarkdown(text string) string {
	var b strings.Builder

	for i := 0; i < len(text); i++ {
		ch := text[i]

		// Escape hyphen only if at start of line (after newline or at beginning)
		if ch == '-' {
			if i == 0 || text[i-1] == '\n' {
				b.WriteString("\\-")
				continue
			}
			b.WriteByte(ch)
			continue
		}

		// Escape other markdown characters
		switch ch {
		case '_':
			b.WriteString("\\_")
		case '*':
			b.WriteString("\\*")
		case '[':
			b.WriteString("\\[")
		case ']':
			b.WriteString("\\]")
		case '~':
			b.WriteString("\\~")
		case '`':
			b.WriteString("\\`")
		case '>':
			b.WriteString("\\>")
		case '#':
			b.WriteString("\\#")
		case '+':
			b.WriteString("\\+")
		case '=':
			b.WriteString("\\=")
		case '|':
			b.WriteString("\\|")
		case '{':
			b.WriteString("\\{")
		case '}':
			b.WriteString("\\}")
		case '!':
			b.WriteString("\\!")
		default:
			b.WriteByte(ch)
		}
	}

	return b.String()
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

	messages := f.Format(data)
	return strings.Join(messages, "\n\n")
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
