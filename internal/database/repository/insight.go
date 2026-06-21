package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Insight represents a learning insight stored in the database.
type Insight struct {
	ID          int             `db:"id" json:"id"`
	Category    string          `db:"category" json:"category"`
	Level       string          `db:"level" json:"level"`
	Title       string          `db:"title" json:"title"`
	Insight     string          `db:"insight" json:"insight"`
	Key         *string         `db:"key" json:"key,omitempty"`
	KeyPoints   pq.StringArray  `db:"key_points" json:"key_points"`
	CodeExample *string         `db:"code_example" json:"code_example,omitempty"`
	FollowUps   json.RawMessage `db:"follow_ups" json:"follow_ups"`
	Tags        pq.StringArray  `db:"tags" json:"tags"`
	TimesSent   int             `db:"times_sent" json:"times_sent"`
	LastSentAt  *time.Time      `db:"last_sent_at" json:"last_sent_at,omitempty"`
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
}

// InsightRepository handles CRUD operations for insights.
type InsightRepository struct {
	db *sqlx.DB
}

// NewInsightRepository creates a new InsightRepository.
func NewInsightRepository(db *sqlx.DB) *InsightRepository {
	return &InsightRepository{db: db}
}

// Create inserts a new insight into the database.
func (r *InsightRepository) Create(ctx context.Context, insight *Insight) error {
	query := `
		INSERT INTO insights (category, level, title, insight, key, key_points, code_example, follow_ups, tags)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`

	err := r.db.QueryRowContext(ctx, query,
		insight.Category,
		insight.Level,
		insight.Title,
		insight.Insight,
		insight.Key,
		insight.KeyPoints,
		insight.CodeExample,
		insight.FollowUps,
		insight.Tags,
	).Scan(&insight.ID, &insight.CreatedAt, &insight.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create insight: %w", err)
	}

	return nil
}

// GetByID retrieves an insight by its ID.
func (r *InsightRepository) GetByID(ctx context.Context, id int) (*Insight, error) {
	query := `
		SELECT id, category, level, title, insight, key, key_points, code_example, follow_ups, tags,
		       times_sent, last_sent_at, created_at, updated_at
		FROM insights WHERE id = $1`

	var insight Insight
	err := r.db.GetContext(ctx, &insight, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get insight by id %d: %w", id, err)
	}

	return &insight, nil
}

// GetByCategoryAndLevel retrieves insights filtered by category and level.
func (r *InsightRepository) GetByCategoryAndLevel(ctx context.Context, category, level string) ([]Insight, error) {
	query := `
		SELECT id, category, level, title, insight, key, key_points, code_example, follow_ups, tags,
		       times_sent, last_sent_at, created_at, updated_at
		FROM insights
		WHERE category = $1 AND level = $2
		ORDER BY last_sent_at ASC NULLS FIRST`

	var insights []Insight
	err := r.db.SelectContext(ctx, &insights, query, category, level)
	if err != nil {
		return nil, fmt.Errorf("failed to get insights by category %s and level %s: %w", category, level, err)
	}

	return insights, nil
}

// GetUnsentInWindow retrieves insights that haven't been sent within the dedup window.
func (r *InsightRepository) GetUnsentInWindow(ctx context.Context, category, level string, windowHours int) ([]Insight, error) {
	query := `
		SELECT i.id, i.category, i.level, i.title, i.insight, i.key, i.key_points, i.code_example,
		       i.follow_ups, i.tags, i.times_sent, i.last_sent_at, i.created_at, i.updated_at
		FROM insights i
		WHERE i.category = $1 AND i.level = $2
		AND (
			i.last_sent_at IS NULL
			OR i.last_sent_at < NOW() - ($3 || ' hours')::INTERVAL
		)
		ORDER BY i.last_sent_at ASC NULLS FIRST
		LIMIT 10`

	var insights []Insight
	err := r.db.SelectContext(ctx, &insights, query, category, level, fmt.Sprintf("%d", windowHours))
	if err != nil {
		return nil, fmt.Errorf("failed to get unsent insights: %w", err)
	}

	return insights, nil
}

// GetRandomByCategoryAndLevel retrieves a random insight for a given category and level.
func (r *InsightRepository) GetRandomByCategoryAndLevel(ctx context.Context, category, level string) (*Insight, error) {
	query := `
		SELECT id, category, level, title, insight, key, key_points, code_example, follow_ups, tags,
		       times_sent, last_sent_at, created_at, updated_at
		FROM insights
		WHERE category = $1 AND level = $2
		ORDER BY RANDOM()
		LIMIT 1`

	var insight Insight
	err := r.db.GetContext(ctx, &insight, query, category, level)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get random insight: %w", err)
	}

	return &insight, nil
}

// UpdateSentStatus updates the times_sent and last_sent_at after delivery.
func (r *InsightRepository) UpdateSentStatus(ctx context.Context, id int) error {
	query := `
		UPDATE insights
		SET times_sent = times_sent + 1, last_sent_at = NOW()
		WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to update sent status for insight %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("insight %d not found", id)
	}

	return nil
}

// CountByCategory returns the count of insights per category.
func (r *InsightRepository) CountByCategory(ctx context.Context) (map[string]int, error) {
	query := `SELECT category, COUNT(*) as count FROM insights GROUP BY category`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to count insights by category: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int)
	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		result[category] = count
	}

	return result, rows.Err()
}

// SearchByTags searches insights by tags.
func (r *InsightRepository) SearchByTags(ctx context.Context, tags []string) ([]Insight, error) {
	query := `
		SELECT id, category, level, title, insight, key, key_points, code_example, follow_ups, tags,
		       times_sent, last_sent_at, created_at, updated_at
		FROM insights
		WHERE tags && $1
		ORDER BY last_sent_at ASC NULLS FIRST`

	var insights []Insight
	err := r.db.SelectContext(ctx, &insights, query, pq.Array(tags))
	if err != nil {
		return nil, fmt.Errorf("failed to search insights by tags: %w", err)
	}

	return insights, nil
}

// Delete removes an insight by ID.
func (r *InsightRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM insights WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete insight %d: %w", id, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("insight %d not found", id)
	}

	return nil
}

// GetByCategoryAndKey retrieves insights by category and key, ordered by last_sent_at.
// This is used for variation generation when we want to create a variation of an existing insight.
func (r *InsightRepository) GetByCategoryAndKey(ctx context.Context, category, key string) ([]Insight, error) {
	query := `
		SELECT id, category, level, title, insight, key, key_points, code_example, follow_ups, tags,
		       times_sent, last_sent_at, created_at, updated_at
		FROM insights
		WHERE category = $1 AND key = $2
		ORDER BY last_sent_at ASC NULLS FIRST
		LIMIT 5`

	var insights []Insight
	err := r.db.SelectContext(ctx, &insights, query, category, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get insights by category %s and key %s: %w", category, key, err)
	}

	return insights, nil
}
