package rotation

import (
	"math"
	"time"
)

// SpacedRepetition implements spaced repetition heuristics for insight scheduling.
type SpacedRepetition struct {
	config EngineConfig
}

// NewSpacedRepetition creates a new SpacedRepetition instance.
func NewSpacedRepetition(config EngineConfig) *SpacedRepetition {
	return &SpacedRepetition{
		config: config,
	}
}

// RepetitionState represents the learning state for a specific topic/category.
type RepetitionState struct {
	Category   string
	Level      string
	TotalSent  int
	LastSentAt time.Time
	EaseFactor float64
	Interval   time.Duration
}

// CalculateNextInterval calculates the next review interval using SM-2 inspired algorithm.
func (sr *SpacedRepetition) CalculateNextInterval(state RepetitionState, quality int) time.Duration {
	// Quality: 0 (worst) to 5 (best)
	// Default ease factor: 2.5
	if state.EaseFactor == 0 {
		state.EaseFactor = 2.5
	}

	// Update ease factor based on quality
	newEF := sr.calculateEaseFactor(state.EaseFactor, quality)

	// Calculate interval
	var newInterval time.Duration

	if quality < 3 {
		// If quality is low, reset interval
		newInterval = time.Duration(sr.config.MinDaysBeforeRepeat) * 24 * time.Hour
	} else if state.TotalSent == 0 {
		// First review
		newInterval = time.Duration(sr.config.MinDaysBeforeRepeat) * 24 * time.Hour
	} else if state.TotalSent == 1 {
		// Second review
		newInterval = time.Duration(float64(sr.config.MinDaysBeforeRepeat)*1.5) * 24 * time.Hour
	} else {
		// Subsequent reviews: multiply by ease factor
		days := float64(state.Interval.Hours()/24) * newEF
		if days < float64(sr.config.MinDaysBeforeRepeat) {
			days = float64(sr.config.MinDaysBeforeRepeat)
		}
		newInterval = time.Duration(days) * 24 * time.Hour
	}

	return newInterval
}

// calculateEaseFactor calculates the new ease factor based on quality.
func (sr *SpacedRepetition) calculateEaseFactor(currentEF float64, quality int) float64 {
	// SM-2 ease factor formula
	newEF := currentEF + (0.1 - float64(5-quality)*(0.08+float64(5-quality)*0.02))

	// Ease factor should not be below 1.3
	if newEF < 1.3 {
		newEF = 1.3
	}

	return newEF
}

// GetLevelProgression determines the next level based on mastery.
func (sr *SpacedRepetition) GetLevelProgression(currentLevel string, timesSent int) string {
	if timesSent < 5 {
		return currentLevel
	}

	switch currentLevel {
	case "beginner":
		if timesSent >= 5 {
			return "intermediate"
		}
	case "intermediate":
		if timesSent >= 15 {
			return "advanced"
		}
	}

	return currentLevel
}

// CalculateMastery calculates a mastery score (0-100) for a category.
func (sr *SpacedRepetition) CalculateMastery(state RepetitionState) float64 {
	if state.TotalSent == 0 {
		return 0
	}

	// Base score from total sent
	baseScore := math.Min(float64(state.TotalSent)*5, 60)

	// Time-based decay
	daysSinceLastReview := time.Since(state.LastSentAt).Hours() / 24
	decay := math.Max(0, 100-daysSinceLastReview*2)

	// Combined score
	mastery := (baseScore + decay) / 2

	return math.Min(mastery, 100)
}

// ShouldUpgrade checks if a category should be upgraded to the next level.
func (sr *SpacedRepetition) ShouldUpgrade(state RepetitionState) bool {
	if state.TotalSent < 5 {
		return false
	}

	mastery := sr.CalculateMastery(state)

	switch state.Level {
	case "beginner":
		return mastery >= 60 && state.TotalSent >= 5
	case "intermediate":
		return mastery >= 70 && state.TotalSent >= 15
	case "advanced":
		return false // Never upgrade from advanced
	}

	return false
}
