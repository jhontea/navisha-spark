package content

import (
	"fmt"
	"strings"
)

// ValidationResult holds the result of content validation.
type ValidationResult struct {
	Valid    bool
	Errors   []string
	Warnings []string
}

// Validator validates generated content for quality and completeness.
type Validator struct {
	minInsightLength int
	maxInsightLength int
	minKeyPoints     int
	maxKeyPoints     int
	minFollowUps     int
	maxFollowUps     int
	minTags          int
	maxTags          int
	maxTitleLength   int
}

// NewValidator creates a new Validator with default thresholds.
func NewValidator() *Validator {
	return &Validator{
		minInsightLength: 100,
		maxInsightLength: 5000,
		minKeyPoints:     2,
		maxKeyPoints:     10,
		minFollowUps:     1,
		maxFollowUps:     5,
		minTags:          1,
		maxTags:          10,
		maxTitleLength:   150,
	}
}

// Validate validates a GeneratedInsight for quality and completeness.
func (v *Validator) Validate(insight *GeneratedInsight) *ValidationResult {
	result := &ValidationResult{
		Valid:    true,
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}

	// Validate title
	v.validateTitle(insight.Title, result)

	// Validate insight content
	v.validateInsightContent(insight.Insight, result)

	// Validate key points
	v.validateKeyPoints(insight.KeyPoints, result)

	// Validate follow-ups
	v.validateFollowUps(insight.FollowUps, result)

	// Validate tags
	v.validateTags(insight.Tags, result)

	// Validate code example if present
	if insight.CodeExample != "" {
		v.validateCodeExample(insight.CodeExample, result)
	}

	result.Valid = len(result.Errors) == 0
	return result
}

// validateTitle checks if the title is valid.
func (v *Validator) validateTitle(title string, result *ValidationResult) {
	if title == "" {
		result.Errors = append(result.Errors, "title is empty")
		return
	}

	if len(title) > v.maxTitleLength {
		result.Errors = append(result.Errors, fmt.Sprintf(
			"title too long: %d characters (max %d)", len(title), v.maxTitleLength))
	}

	if len(title) < 5 {
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"title is very short: %d characters", len(title)))
	}
}

// validateInsightContent checks if the insight content is valid.
func (v *Validator) validateInsightContent(content string, result *ValidationResult) {
	if content == "" {
		result.Errors = append(result.Errors, "insight content is empty")
		return
	}

	if len(content) < v.minInsightLength {
		result.Errors = append(result.Errors, fmt.Sprintf(
			"insight too short: %d characters (min %d)", len(content), v.minInsightLength))
	}

	if len(content) > v.maxInsightLength {
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"insight is very long: %d characters (max %d)", len(content), v.maxInsightLength))
	}
}

// validateKeyPoints checks if the key points are valid.
func (v *Validator) validateKeyPoints(points []string, result *ValidationResult) {
	if len(points) < v.minKeyPoints {
		result.Errors = append(result.Errors, fmt.Sprintf(
			"not enough key points: %d (min %d)", len(points), v.minKeyPoints))
	}

	if len(points) > v.maxKeyPoints {
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"too many key points: %d (max %d)", len(points), v.maxKeyPoints))
	}

	for i, point := range points {
		if strings.TrimSpace(point) == "" {
			result.Errors = append(result.Errors, fmt.Sprintf(
				"key point %d is empty", i+1))
		}
	}
}

// validateFollowUps checks if follow-up questions are valid.
func (v *Validator) validateFollowUps(followUps []FollowUp, result *ValidationResult) {
	if len(followUps) < v.minFollowUps {
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"no follow-up questions (recommended min %d)", v.minFollowUps))
	}

	if len(followUps) > v.maxFollowUps {
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"too many follow-ups: %d (max %d)", len(followUps), v.maxFollowUps))
	}

	for i, fu := range followUps {
		if strings.TrimSpace(fu.Question) == "" {
			result.Errors = append(result.Errors, fmt.Sprintf(
				"follow-up %d has empty question", i+1))
		}
		if strings.TrimSpace(fu.Answer) == "" {
			result.Errors = append(result.Errors, fmt.Sprintf(
				"follow-up %d has empty answer", i+1))
		}
	}
}

// validateTags checks if the tags are valid.
func (v *Validator) validateTags(tags []string, result *ValidationResult) {
	if len(tags) < v.minTags {
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"not enough tags: %d (recommended min %d)", len(tags), v.minTags))
	}

	if len(tags) > v.maxTags {
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"too many tags: %d (max %d)", len(tags), v.maxTags))
	}

	for i, tag := range tags {
		if strings.TrimSpace(tag) == "" {
			result.Errors = append(result.Errors, fmt.Sprintf(
				"tag %d is empty", i+1))
		}
	}
}

// validateCodeExample checks if the code example looks valid.
func (v *Validator) validateCodeExample(code string, result *ValidationResult) {
	if len(code) < 10 {
		result.Warnings = append(result.Warnings, "code example is very short")
	}

	if len(code) > 2000 {
		result.Warnings = append(result.Warnings, fmt.Sprintf(
			"code example is very long: %d characters", len(code)))
	}
}

// ValidateAndRepair attempts to fix common issues in generated content.
func (v *Validator) ValidateAndRepair(insight *GeneratedInsight) (*GeneratedInsight, *ValidationResult) {
	result := v.Validate(insight)

	// Trim whitespace from all string fields
	insight.Title = strings.TrimSpace(insight.Title)
	insight.Insight = strings.TrimSpace(insight.Insight)
	insight.CodeExample = strings.TrimSpace(insight.CodeExample)

	// Clean up key points
	cleanedPoints := make([]string, 0)
	for _, p := range insight.KeyPoints {
		p = strings.TrimSpace(p)
		if p != "" {
			// Remove leading bullet points if present
			p = strings.TrimPrefix(p, "- ")
			p = strings.TrimPrefix(p, "* ")
			p = strings.TrimPrefix(p, "• ")
			cleanedPoints = append(cleanedPoints, p)
		}
	}
	insight.KeyPoints = cleanedPoints

	// Clean up tags
	cleanedTags := make([]string, 0)
	for _, t := range insight.Tags {
		t = strings.TrimSpace(t)
		if t != "" {
			cleanedTags = append(cleanedTags, strings.ToLower(t))
		}
	}
	insight.Tags = cleanedTags

	return insight, result
}

// IsContentAcceptable checks if the content meets minimum quality standards.
func (v *Validator) IsContentAcceptable(result *ValidationResult) bool {
	if !result.Valid {
		return false
	}

	// If there are too many warnings, consider it unacceptable
	if len(result.Warnings) > 5 {
		return false
	}

	return true
}
