package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// SentHistory represents a record of a sent insight.
type SentHistory struct {
	InsightID int       `db:"insight_id" json:"insight_id"`
	SentAt    time.Time `db:"sent_at" json:"sent_at"`
}

// HistoryRepository handles deduplication checks and sent history.
type HistoryRepository struct {
	db *sqlx.DB
}

// NewHistoryRepository creates a new HistoryRepository.
func NewHistoryRepository(db *sqlx.DB) *HistoryRepository {
	return &HistoryRepository{db: db}
}

// Record inserts a new sent history record.
func (r *HistoryRepository) Record(ctx context.Context, insightID int) error {
	query := `INSERT INTO sent_history (insight_id) VALUES ($1)`

	_, err := r.db.ExecContext(ctx, query, insightID)
	if err != nil {
		return fmt.Errorf("failed to record sent history for insight %d: %w", insightID, err)
	}

	return nil
}

// IsDuplicate checks if an insight was sent within the specified window.
func (r *HistoryRepository) IsDuplicate(ctx context.Context, insightID int, windowHours int) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM sent_history
			WHERE insight_id = $1
			AND sent_at > NOW() - ($2 * INTERVAL '1 hour')
		)`

	var exists bool
	err := r.db.QueryRowContext(ctx, query, insightID, windowHours).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check duplicate for insight %d: %w", insightID, err)
	}

	return exists, nil
}

// GetRecentByInsightID retrieves recent sent history for an insight.
func (r *HistoryRepository) GetRecentByInsightID(ctx context.Context, insightID int, limit int) ([]SentHistory, error) {
	query := `
		SELECT insight_id, sent_at
		FROM sent_history
		WHERE insight_id = $1
		ORDER BY sent_at DESC
		LIMIT $2`

	var history []SentHistory
	err := r.db.SelectContext(ctx, &history, query, insightID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get sent history for insight %d: %w", insightID, err)
	}

	return history, nil
}

// GetSentInsightIDsInWindow returns all insight IDs that were sent within the window.
func (r *HistoryRepository) GetSentInsightIDsInWindow(ctx context.Context, windowHours int) ([]int, error) {
	query := `
		SELECT DISTINCT insight_id FROM sent_history
		WHERE sent_at > NOW() - ($1 * INTERVAL '1 hour')`

	var ids []int
	err := r.db.SelectContext(ctx, &ids, query, windowHours)
	if err != nil {
		return nil, fmt.Errorf("failed to get sent insight ids in window: %w", err)
	}

	return ids, nil
}

// CleanupOld removes sent history older than the specified days.
func (r *HistoryRepository) CleanupOld(ctx context.Context, days int) (int, error) {
	query := `DELETE FROM sent_history WHERE sent_at < NOW() - ($1 * INTERVAL '1 day')`

	result, err := r.db.ExecContext(ctx, query, days)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old sent history: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rows), nil
}

// CountInWindow returns the count of distinct insights sent within the window.
func (r *HistoryRepository) CountInWindow(ctx context.Context, windowHours int) (int, error) {
	query := `
		SELECT COUNT(DISTINCT insight_id) FROM sent_history
		WHERE sent_at > NOW() - ($1 * INTERVAL '1 hour')`

	var count int
	err := r.db.QueryRowContext(ctx, query, windowHours).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count sent insights in window: %w", err)
	}

	return count, nil
}
