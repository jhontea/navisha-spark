package rotation

import (
	"context"
	"fmt"
	"time"

	"github.com/navisha/spark/internal/database/repository"
)

// Deduplicator handles deduplication logic to prevent sending the same content.
type Deduplicator struct {
	insightRepo *repository.InsightRepository
	historyRepo *repository.HistoryRepository
	config      EngineConfig
}

// NewDeduplicator creates a new Deduplicator.
func NewDeduplicator(
	insightRepo *repository.InsightRepository,
	historyRepo *repository.HistoryRepository,
	config EngineConfig,
) *Deduplicator {
	return &Deduplicator{
		insightRepo: insightRepo,
		historyRepo: historyRepo,
		config:      config,
	}
}

// IsDuplicate checks if an insight has been sent within the dedup window.
func (d *Deduplicator) IsDuplicate(ctx context.Context, insightID int) (bool, error) {
	return d.historyRepo.IsDuplicate(ctx, insightID, d.config.DedupWindowHours)
}

// FilterDuplicates filters out insights that have been sent recently.
func (d *Deduplicator) FilterDuplicates(ctx context.Context, insights []repository.Insight) ([]repository.Insight, error) {
	if len(insights) == 0 {
		return insights, nil
	}

	var filtered []repository.Insight
	for _, insight := range insights {
		isDup, err := d.IsDuplicate(ctx, insight.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check duplicate for insight %d: %w", insight.ID, err)
		}
		if !isDup {
			filtered = append(filtered, insight)
		}
	}

	return filtered, nil
}

// GetAvailableIDs returns insight IDs that are available to send (not in dedup window).
func (d *Deduplicator) GetAvailableIDs(ctx context.Context, category string, level string) ([]int, error) {
	insights, err := d.insightRepo.GetByCategoryAndLevel(ctx, category, level)
	if err != nil {
		return nil, fmt.Errorf("failed to get insights: %w", err)
	}

	// Get sent IDs in window
	sentIDs, err := d.historyRepo.GetSentInsightIDsInWindow(ctx, d.config.DedupWindowHours)
	if err != nil {
		return nil, fmt.Errorf("failed to get sent IDs: %w", err)
	}

	sentMap := make(map[int]bool)
	for _, id := range sentIDs {
		sentMap[id] = true
	}

	var available []int
	for _, insight := range insights {
		if !sentMap[insight.ID] {
			available = append(available, insight.ID)
		}
	}

	return available, nil
}

// CountAvailable returns the count of available insights for a category and level.
func (d *Deduplicator) CountAvailable(ctx context.Context, category, level string) (int, error) {
	ids, err := d.GetAvailableIDs(ctx, category, level)
	if err != nil {
		return 0, err
	}
	return len(ids), nil
}

// GetNextSendTime calculates when this insight can be sent again.
func (d *Deduplicator) GetNextSendTime(insight *repository.Insight) time.Time {
	if insight.LastSentAt == nil {
		return time.Now()
	}

	// Base repeat interval: min_days_before_repeat
	baseInterval := time.Duration(d.config.MinDaysBeforeRepeat) * 24 * time.Hour

	// Increase interval for insights that have been sent many times
	if insight.TimesSent > 3 {
		multiplier := 1.0 + float64(insight.TimesSent)*0.5
		baseInterval = time.Duration(float64(baseInterval) * multiplier)
	}

	return insight.LastSentAt.Add(baseInterval)
}

// CanSendAgain checks if an insight can be sent again based on repeat interval.
func (d *Deduplicator) CanSendAgain(insight *repository.Insight) bool {
	if insight.LastSentAt == nil {
		return true
	}

	nextSend := d.GetNextSendTime(insight)
	return time.Now().After(nextSend)
}
