package telegram

import (
	"strings"
	"testing"
)

func TestFormatParagraphs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single paragraph",
			input:    "This is a single paragraph.",
			expected: "This is a single paragraph.",
		},
		{
			name:     "multiple paragraphs with double newlines",
			input:    "First paragraph.\n\nSecond paragraph.\n\nThird paragraph.",
			expected: "First paragraph.\n\nSecond paragraph.\n\nThird paragraph.",
		},
		{
			name:     "paragraphs with excessive whitespace",
			input:    "First paragraph.\n\n\n\nSecond paragraph.",
			expected: "First paragraph.\n\nSecond paragraph.",
		},
		{
			name:     "paragraphs with leading/trailing whitespace",
			input:    "  First paragraph.  \n\n  Second paragraph.  ",
			expected: "First paragraph.\n\nSecond paragraph.",
		},
		{
			name:     "empty paragraphs",
			input:    "First paragraph.\n\n\n\n\nSecond paragraph.",
			expected: "First paragraph.\n\nSecond paragraph.",
		},
		{
			name:     "markdown special characters",
			input:    "This is *bold* and _italic_.\n\nThis is a new paragraph with [link](url).",
			expected: "This is \\*bold\\* and \\_italic\\_.\n\nThis is a new paragraph with \\[link\\](url).",
		},
		{
			name:     "hyphens in words should not be escaped",
			input:    "Fine-tuning is important.\n\nUse 1e-5 learning rate.",
			expected: "Fine-tuning is important.\n\nUse 1e-5 learning rate.",
		},
		{
			name:     "pre-escaped parentheses should be unescaped",
			input:    "Use \\(Set If Not eXists\\) command\n\nThis is a new paragraph.",
			expected: "Use (Set If Not eXists) command\n\nThis is a new paragraph.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatParagraphs(tt.input)
			if result != tt.expected {
				t.Errorf("formatParagraphs() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFormatWithParagraphs(t *testing.T) {
	formatter := NewFormatter(FormatConfig{
		IncludeCategory:  true,
		IncludeLevel:     true,
		IncludeFollowUps: true,
		IncludeTags:      true,
	})

	data := &InsightData{
		Category: "Golang",
		Level:    "intermediate",
		Title:    "Understanding Goroutines",
		Insight:  "Goroutines are lightweight threads managed by the Go runtime.\n\nThey are cheaper than OS threads and allow you to run thousands of concurrent tasks.\n\nChannels provide a way for goroutines to communicate safely.",
		KeyPoints: []string{
			"Goroutines use ~2KB stack initially",
			"Channels enable safe communication",
			"Select statement handles multiple channels",
		},
		FollowUps: []FollowUp{
			{Question: "What is the difference between goroutine and thread?", Answer: "Goroutines are managed by Go runtime, not OS."},
		},
		Tags: []string{"golang", "concurrency", "goroutine"},
	}

	result := formatter.Format(data)
	resultStr := strings.Join(result, "\n\n")

	// Verify the result contains proper paragraph breaks
	if !strings.Contains(resultStr, "Goroutines are lightweight threads managed by the Go runtime.\n\nThey are cheaper than OS threads") {
		t.Error("Expected proper paragraph breaks in insight text")
	}

	// Verify markdown is escaped
	if strings.Contains(resultStr, "*bold*") {
		t.Error("Markdown should be escaped")
	}

	// Verify structure is maintained
	if !strings.Contains(resultStr, "📚 *Golang* — *🟡 Intermediate*") {
		t.Error("Header should contain category and level")
	}

	if !strings.Contains(resultStr, "*Understanding Goroutines*") {
		t.Error("Title should be present")
	}

	if !strings.Contains(resultStr, "💡 *Key Points:*") {
		t.Error("Key points section should be present")
	}

	if !strings.Contains(resultStr, "🔍 *Deep Dive:*") {
		t.Error("Deep dive section should be present")
	}

	if !strings.Contains(resultStr, "---\n_Tags: golang, concurrency, goroutine_") {
		t.Error("Tags section should be present")
	}
}

func TestFormatWithLongInsight(t *testing.T) {
	formatter := NewFormatter(FormatConfig{
		IncludeCategory: true,
		IncludeLevel:    true,
	})

	longInsight := `Concurrency adalah salah satu kekuatan utama Go yang membedakannya dari bahasa pemrograman lain.

Dengan goroutine, channel, dan select statement, Go menyediakan model concurrent programming yang elegan dan aman.

Goroutine adalah thread ringan yang dikelola oleh Go runtime, memungkinkan eksekusi ribuan task secara paralel tanpa overhead thread OS yang berat.

Channel berfungsi sebagai mekanisme komunikasi antar goroutine yang aman, mencegah data race dan memudahkan sinkronisasi.

Select statement memungkinkan goroutine menunggu multiple channel operations secara bersamaan, mirip switch case tetapi untuk channel.

Pola ini sangat berguna untuk timeout handling, fan-in/fan-out, dan pipeline processing.

Dalam pengembangan backend, concurrency di Go sangat efektif untuk menangani request HTTP secara paralel, database query, API calls ke multiple services, dan message processing.

Namun, penggunaan concurrency yang berlebihan tanpa proper synchronization dapat menyebabkan deadlock, goroutine leak, dan data race.

Best practice utama selalu menggunakan channel untuk komunikasi, hindari shared memory, dan gunakan context untuk cancellation.

Untuk kasus sederhana, sequential processing lebih mudah di-maintain. Concurrency sebaiknya digunakan ketika ada I/O bound operations atau independent tasks yang bisa dieksekusi paralel.`

	data := &InsightData{
		Category: "Golang",
		Level:    "intermediate",
		Title:    "Go Concurrency: Goroutine, Channel, dan Select",
		Insight:  longInsight,
	}

	result := formatter.Format(data)
	resultStr := strings.Join(result, "\n\n")

	// Count the number of double newlines in the insight section
	insightSection := strings.Split(resultStr, "📝 *Insight:*\n")[1]
	insightSection = strings.Split(insightSection, "\n\n💡")[0]

	paragraphCount := strings.Count(insightSection, "\n\n")
	if paragraphCount < 5 {
		t.Errorf("Expected at least 5 paragraph breaks, got %d", paragraphCount)
	}

	// Verify no excessive whitespace
	if strings.Contains(insightSection, "\n\n\n") {
		t.Error("Should not contain triple newlines")
	}
}
