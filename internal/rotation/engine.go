package rotation

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/navisha/spark/internal/database/repository"
)

// Engine handles the main rotation logic for selecting insights.
type Engine struct {
	rotationRepo *repository.RotationRepository
	insightRepo  *repository.InsightRepository
	historyRepo  *repository.HistoryRepository
	selector     *Selector
	config       EngineConfig
	log          *logrus.Entry
}

// EngineConfig holds configuration for the rotation engine.
type EngineConfig struct {
	LevelDistribution   map[string]int
	MinDaysBeforeRepeat int
	DedupWindowHours    int
	DefaultWeight       float64
}

// SelectionResult represents the result of a rotation selection.
type SelectionResult struct {
	Insight  *repository.Insight
	Category string
	Level    string
	Key      string
	FromBank bool
}

// NewEngine creates a new rotation engine.
func NewEngine(
	rotationRepo *repository.RotationRepository,
	insightRepo *repository.InsightRepository,
	historyRepo *repository.HistoryRepository,
	config EngineConfig,
	log *logrus.Entry,
) *Engine {
	return &Engine{
		rotationRepo: rotationRepo,
		insightRepo:  insightRepo,
		historyRepo:  historyRepo,
		selector:     NewSelector(config),
		config:       config,
		log:          log,
	}
}

// SelectNext selects the next insight to send based on rotation logic.
func (e *Engine) SelectNext(ctx context.Context, categories []string) (*SelectionResult, error) {
	// Get current rotation states
	states, err := e.rotationRepo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get rotation states: %w", err)
	}

	// Build state map
	stateMap := make(map[string]*repository.RotationState)
	for i := range states {
		stateMap[states[i].Category] = &states[i]
	}

	// Calculate priorities for each category
	type categoryPriority struct {
		category string
		priority float64
		state    *repository.RotationState
	}

	var priorities []categoryPriority
	for _, cat := range categories {
		state := stateMap[cat]
		priority := e.calculatePriority(cat, state)
		priorities = append(priorities, categoryPriority{
			category: cat,
			priority: priority,
			state:    state,
		})
	}

	// Sort by priority (highest first)
	sort.Slice(priorities, func(i, j int) bool {
		return priorities[i].priority > priorities[j].priority
	})

	// Try to select an insight from the highest priority categories
	for _, cp := range priorities {
		// Determine level based on spaced repetition heuristic
		level := e.selector.SelectLevel(cp.state)

		// Try to get an unsent insight from the bank
		insight, err := e.insightRepo.GetUnsentInWindow(ctx, cp.category, level, e.config.DedupWindowHours)
		if err != nil {
			e.log.WithFields(logrus.Fields{
				"category": cp.category,
				"level":    level,
				"error":    err,
			}).Warn("failed to get unsent insights")
			continue
		}

		if len(insight) > 0 {
			// Found an insight in the bank
			selectedInsight := insight[0]

			e.log.WithFields(logrus.Fields{
				"category":   cp.category,
				"level":      level,
				"insight_id": selectedInsight.ID,
				"key":        selectedInsight.Key,
				"source":     "bank",
			}).Info("selected insight from bank")

			key := ""
			if selectedInsight.Key != nil {
				key = *selectedInsight.Key
			}

			return &SelectionResult{
				Insight:  &selectedInsight,
				Category: cp.category,
				Level:    level,
				Key:      key,
				FromBank: true,
			}, nil
		}

		// No unsent insight found, check if we should generate a variation
		// Pick a random key from subtopics for this category
		key := e.pickRandomKey(cp.category)
		if key != "" {
			// Check if there are existing insights with this key
			existingInsights, err := e.insightRepo.GetByCategoryAndKey(ctx, cp.category, key)
			if err == nil && len(existingInsights) > 0 {
				// Found existing insights with this key, select the oldest one for variation
				baseInsight := existingInsights[0]

				e.log.WithFields(logrus.Fields{
					"category": cp.category,
					"level":    level,
					"key":      key,
					"base_id":  baseInsight.ID,
					"source":   "variation",
				}).Info("will generate variation for existing key")

				return &SelectionResult{
					Insight:  &baseInsight,
					Category: cp.category,
					Level:    level,
					Key:      key,
					FromBank: false, // Mark as variation needed
				}, nil
			}
		}

		// No insight found in bank for this category, try next category
		e.log.WithFields(logrus.Fields{
			"category": cp.category,
			"level":    level,
		}).Debug("no unsent insights in bank for category")
	}

	// If no insights found in any category, return nil to trigger LLM generation
	e.log.Warn("no insights found in bank for any category, will need LLM generation")
	return nil, nil
}

// pickRandomKey picks a random subtopic key for a given category.
// This is used when we need to generate a variation or new insight.
func (e *Engine) pickRandomKey(category string) string {
	// This is a simplified version - in production, this would use the category config
	// For now, we'll return a generic key based on the category
	// The actual subtopic selection is done in the scheduler job
	return ""
}

// calculatePriority calculates the priority score for a category.
func (e *Engine) calculatePriority(category string, state *repository.RotationState) float64 {
	now := time.Now()

	if state == nil || state.LastSentAt == nil {
		// Category has never been sent, give it maximum priority
		return math.MaxFloat64
	}

	// Time since last sent (in hours)
	hoursSinceLastSent := now.Sub(*state.LastSentAt).Hours()

	// Base priority is the time since last sent
	priority := hoursSinceLastSent

	// Apply weight (default 1.0)
	priority *= e.config.DefaultWeight

	// Bonus for categories that have been sent less frequently
	if state.TotalSent > 0 {
		avgInterval := hoursSinceLastSent / float64(state.TotalSent)
		if avgInterval > 24 { // If average interval is more than 24 hours
			priority *= 1.5 // Boost priority
		}
	}

	return priority
}

// RecordDelivery records a successful delivery and updates rotation state.
func (e *Engine) RecordDelivery(ctx context.Context, category, level string, insightID int) error {
	// Update rotation state
	if err := e.rotationRepo.UpdateSent(ctx, category, level); err != nil {
		return fmt.Errorf("failed to update rotation state: %w", err)
	}

	// Update insight sent status
	if err := e.insightRepo.UpdateSentStatus(ctx, insightID); err != nil {
		return fmt.Errorf("failed to update insight sent status: %w", err)
	}

	// Record in sent history
	if err := e.historyRepo.Record(ctx, insightID); err != nil {
		return fmt.Errorf("failed to record sent history: %w", err)
	}

	return nil
}

// GetCategoryPriority returns the priority for a specific category.
func (e *Engine) GetCategoryPriority(ctx context.Context, category string) (float64, error) {
	state, err := e.rotationRepo.GetByCategory(ctx, category)
	if err != nil {
		return 0, fmt.Errorf("failed to get rotation state for %s: %w", category, err)
	}

	return e.calculatePriority(category, state), nil
}

// GetAllRotationStates returns all rotation states.
func (e *Engine) GetAllRotationStates(ctx context.Context) ([]repository.RotationState, error) {
	return e.rotationRepo.GetAll(ctx)
}

// SelectCategoryForLLM selects the highest priority category for LLM generation.
// This is used when the insight bank has no available content.
func (e *Engine) SelectCategoryForLLM(ctx context.Context, categories []string) (string, error) {
	if len(categories) == 0 {
		return "", fmt.Errorf("no categories available")
	}

	if len(categories) == 1 {
		return categories[0], nil
	}

	// Get all rotation states
	states, err := e.rotationRepo.GetAll(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get rotation states: %w", err)
	}

	// Build state map
	stateMap := make(map[string]*repository.RotationState)
	for i := range states {
		stateMap[states[i].Category] = &states[i]
	}

	// Calculate priorities for each category
	type categoryPriority struct {
		category string
		priority float64
	}

	var priorities []categoryPriority
	for _, cat := range categories {
		state := stateMap[cat]
		priority := e.calculatePriority(cat, state)
		priorities = append(priorities, categoryPriority{
			category: cat,
			priority: priority,
		})
	}

	// Sort by priority (highest first)
	sort.Slice(priorities, func(i, j int) bool {
		return priorities[i].priority > priorities[j].priority
	})

	// Use the selector's weighted random for tie-breaking among top categories
	if len(priorities) > 1 && priorities[0].priority == priorities[1].priority {
		// Top two have same priority, pick randomly between them
		if e.selector.rng.Float64() < 0.5 {
			return priorities[0].category, nil
		}
		return priorities[1].category, nil
	}

	return priorities[0].category, nil
}
