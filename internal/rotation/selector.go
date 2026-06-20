package rotation

import (
	"math/rand"
	"time"

	"github.com/navisha/spark/internal/database/repository"
)

// Selector handles weighted round-robin selection and level distribution.
type Selector struct {
	config EngineConfig
	rng    *rand.Rand
}

// NewSelector creates a new Selector.
func NewSelector(config EngineConfig) *Selector {
	return &Selector{
		config: config,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// SelectLevel selects a difficulty level based on spaced repetition heuristics.
func (s *Selector) SelectLevel(state *repository.RotationState) string {
	// If no state or no previous level, use distribution
	if state == nil || state.LastLevel == nil {
		return s.selectLevelFromDistribution()
	}

	// If total sent is low, prefer beginner/intermediate
	if state.TotalSent < 5 {
		return s.selectLevelFromDistribution()
	}

	// Based on total sent, gradually increase difficulty
	switch {
	case state.TotalSent < 10:
		// Mostly beginner, some intermediate
		return s.weightedRandom(map[string]float64{
			"beginner":     0.4,
			"intermediate": 0.5,
			"advanced":     0.1,
		})
	case state.TotalSent < 20:
		// Mostly intermediate, some advanced
		return s.weightedRandom(map[string]float64{
			"beginner":     0.1,
			"intermediate": 0.6,
			"advanced":     0.3,
		})
	default:
		// Use configured distribution
		return s.selectLevelFromDistribution()
	}
}

// selectLevelFromDistribution selects a level based on the configured distribution.
func (s *Selector) selectLevelFromDistribution() string {
	distribution := s.config.LevelDistribution
	if distribution == nil || len(distribution) == 0 {
		// Default distribution
		distribution = map[string]int{
			"beginner":     20,
			"intermediate": 50,
			"advanced":     30,
		}
	}

	return s.weightedRandomFromInt(distribution)
}

// weightedRandom selects a key from the map based on float64 weights.
func (s *Selector) weightedRandom(weights map[string]float64) string {
	total := 0.0
	for _, w := range weights {
		total += w
	}

	r := s.rng.Float64() * total
	cumulative := 0.0

	// Iterate in consistent order (beginner, intermediate, advanced)
	levels := []string{"beginner", "intermediate", "advanced"}
	for _, level := range levels {
		if w, ok := weights[level]; ok {
			cumulative += w
			if r <= cumulative {
				return level
			}
		}
	}

	return "intermediate" // fallback
}

// weightedRandomFromInt selects a key from the map based on int weights.
func (s *Selector) weightedRandomFromInt(weights map[string]int) string {
	total := 0
	for _, w := range weights {
		total += w
	}

	if total == 0 {
		return "intermediate" // fallback
	}

	r := s.rng.Intn(total)
	cumulative := 0

	// Iterate in consistent order
	levels := []string{"beginner", "intermediate", "advanced"}
	for _, level := range levels {
		if w, ok := weights[level]; ok {
			cumulative += w
			if r < cumulative {
				return level
			}
		}
	}

	return "intermediate" // fallback
}

// SelectCategory selects a category based on weighted round-robin.
func (s *Selector) SelectCategory(categories []string, priorities map[string]float64) string {
	if len(categories) == 0 {
		return ""
	}

	if len(categories) == 1 {
		return categories[0]
	}

	// Calculate total priority
	totalPriority := 0.0
	for _, cat := range categories {
		totalPriority += priorities[cat]
	}

	if totalPriority == 0 {
		// All priorities are zero, pick randomly
		return categories[s.rng.Intn(len(categories))]
	}

	// Weighted random selection
	r := s.rng.Float64() * totalPriority
	cumulative := 0.0

	for _, cat := range categories {
		cumulative += priorities[cat]
		if r <= cumulative {
			return cat
		}
	}

	return categories[len(categories)-1] // fallback
}
