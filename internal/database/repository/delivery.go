package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// DeliveryStatus represents the status of a delivery attempt.
type DeliveryStatus string

const (
	DeliveryStatusSuccess DeliveryStatus = "success"
	DeliveryStatusFailed  DeliveryStatus = "failed"
	DeliveryStatusRetry   DeliveryStatus = "retry"
)

// DeliveryLog represents a delivery log entry.
type DeliveryLog struct {
	ID                int            `db:"id" json:"id"`
	InsightID         int            `db:"insight_id" json:"insight_id"`
	SentAt            time.Time      `db:"sent_at" json:"sent_at"`
	Status            DeliveryStatus `db:"status" json:"status"`
	ErrorMessage      *string        `db:"error_message" json:"error_message,omitempty"`
	TelegramMessageID *int64         `db:"telegram_message_id" json:"telegram_message_id,omitempty"`
}

// DeliveryRepository handles CRUD operations for delivery logs.
type DeliveryRepository struct {
	db *sqlx.DB
}

// NewDeliveryRepository creates a new DeliveryRepository.
func NewDeliveryRepository(db *sqlx.DB) *DeliveryRepository {
	return &DeliveryRepository{db: db}
}

// Create inserts a new delivery log entry.
func (r *DeliveryRepository) Create(ctx context.Context, log *DeliveryLog) error {
	query := `
		INSERT INTO delivery_log (insight_id, status, error_message, telegram_message_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, sent_at`

	err := r.db.QueryRowContext(ctx, query,
		log.InsightID,
		log.Status,
		log.ErrorMessage,
		log.TelegramMessageID,
	).Scan(&log.ID, &log.SentAt)

	if err != nil {
		return fmt.Errorf("failed to create delivery log: %w", err)
	}

	return nil
}

// GetByID retrieves a delivery log by ID.
func (r *DeliveryRepository) GetByID(ctx context.Context, id int) (*DeliveryLog, error) {
	query := `
		SELECT id, insight_id, sent_at, status, error_message, telegram_message_id
		FROM delivery_log WHERE id = $1`

	var log DeliveryLog
	err := r.db.GetContext(ctx, &log, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery log by id %d: %w", id, err)
	}

	return &log, nil
}

// GetByInsightID retrieves all delivery logs for a specific insight.
func (r *DeliveryRepository) GetByInsightID(ctx context.Context, insightID int) ([]DeliveryLog, error) {
	query := `
		SELECT id, insight_id, sent_at, status, error_message, telegram_message_id
		FROM delivery_log
		WHERE insight_id = $1
		ORDER BY sent_at DESC`

	var logs []DeliveryLog
	err := r.db.SelectContext(ctx, &logs, query, insightID)
	if err != nil {
		return nil, fmt.Errorf("failed to get delivery logs for insight %d: %w", insightID, err)
	}

	return logs, nil
}

// GetRecentByStatus retrieves recent delivery logs by status.
func (r *DeliveryRepository) GetRecentByStatus(ctx context.Context, status DeliveryStatus, limit int) ([]DeliveryLog, error) {
	query := `
		SELECT id, insight_id, sent_at, status, error_message, telegram_message_id
		FROM delivery_log
		WHERE status = $1
		ORDER BY sent_at DESC
		LIMIT $2`

	var logs []DeliveryLog
	err := r.db.SelectContext(ctx, &logs, query, status, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent delivery logs by status %s: %w", status, err)
	}

	return logs, nil
}

// GetLatestByInsightID retrieves the latest delivery log for an insight.
func (r *DeliveryRepository) GetLatestByInsightID(ctx context.Context, insightID int) (*DeliveryLog, error) {
	query := `
		SELECT id, insight_id, sent_at, status, error_message, telegram_message_id
		FROM delivery_log
		WHERE insight_id = $1
		ORDER BY sent_at DESC
		LIMIT 1`

	var log DeliveryLog
	err := r.db.GetContext(ctx, &log, query, insightID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest delivery log for insight %d: %w", insightID, err)
	}

	return &log, nil
}

// CountByStatus returns the count of delivery logs grouped by status.
func (r *DeliveryRepository) CountByStatus(ctx context.Context) (map[DeliveryStatus]int, error) {
	query := `SELECT status, COUNT(*) as count FROM delivery_log GROUP BY status`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count delivery logs by status: %w", err)
	}
	defer rows.Close()

	result := make(map[DeliveryStatus]int)
	for rows.Next() {
		var status DeliveryStatus
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		result[status] = count
	}

	return result, rows.Err()
}

// GetRecentDeliveries retrieves the most recent delivery logs.
func (r *DeliveryRepository) GetRecentDeliveries(ctx context.Context, limit int) ([]DeliveryLog, error) {
	query := `
		SELECT id, insight_id, sent_at, status, error_message, telegram_message_id
		FROM delivery_log
		ORDER BY sent_at DESC
		LIMIT $1`

	var logs []DeliveryLog
	err := r.db.SelectContext(ctx, &logs, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent deliveries: %w", err)
	}

	return logs, nil
}
