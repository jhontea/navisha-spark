package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// RotationState represents the rotation state for a category.
type RotationState struct {
	Category   string     `db:"category" json:"category"`
	LastSentAt *time.Time `db:"last_sent_at" json:"last_sent_at,omitempty"`
	TotalSent  int        `db:"total_sent" json:"total_sent"`
	LastLevel  *string    `db:"last_level" json:"last_level,omitempty"`
	UpdatedAt  time.Time  `db:"updated_at" json:"updated_at"`
}

// RotationRepository handles CRUD operations for rotation state.
type RotationRepository struct {
	db *sqlx.DB
}

// NewRotationRepository creates a new RotationRepository.
func NewRotationRepository(db *sqlx.DB) *RotationRepository {
	return &RotationRepository{db: db}
}

// Upsert creates or updates a rotation state entry.
func (r *RotationRepository) Upsert(ctx context.Context, state *RotationState) error {
	query := `
		INSERT INTO rotation_state (category, last_sent_at, total_sent, last_level)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (category)
		DO UPDATE SET
			last_sent_at = EXCLUDED.last_sent_at,
			total_sent = EXCLUDED.total_sent,
			last_level = EXCLUDED.last_level,
			updated_at = NOW()
		RETURNING updated_at`

	err := r.db.QueryRowContext(ctx, query,
		state.Category,
		state.LastSentAt,
		state.TotalSent,
		state.LastLevel,
	).Scan(&state.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to upsert rotation state for %s: %w", state.Category, err)
	}

	return nil
}

// GetByCategory retrieves rotation state for a specific category.
func (r *RotationRepository) GetByCategory(ctx context.Context, category string) (*RotationState, error) {
	query := `
		SELECT category, last_sent_at, total_sent, last_level, updated_at
		FROM rotation_state WHERE category = $1`

	var state RotationState
	err := r.db.GetContext(ctx, &state, query, category)
	if err != nil {
		return nil, fmt.Errorf("failed to get rotation state for %s: %w", category, err)
	}

	return &state, nil
}

// GetAll retrieves all rotation states.
func (r *RotationRepository) GetAll(ctx context.Context) ([]RotationState, error) {
	query := `
		SELECT category, last_sent_at, total_sent, last_level, updated_at
		FROM rotation_state
		ORDER BY category`

	var states []RotationState
	err := r.db.SelectContext(ctx, &states, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all rotation states: %w", err)
	}

	return states, nil
}

// GetStaleCategories retrieves categories that haven't been sent recently,
// ordered by last_sent_at ascending (oldest first).
func (r *RotationRepository) GetStaleCategories(ctx context.Context, limit int) ([]RotationState, error) {
	query := `
		SELECT category, last_sent_at, total_sent, last_level, updated_at
		FROM rotation_state
		ORDER BY last_sent_at ASC NULLS FIRST
		LIMIT $1`

	var states []RotationState
	err := r.db.SelectContext(ctx, &states, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get stale categories: %w", err)
	}

	return states, nil
}

// UpdateSent updates the rotation state after a successful delivery.
func (r *RotationRepository) UpdateSent(ctx context.Context, category, level string) error {
	query := `
		UPDATE rotation_state
		SET last_sent_at = NOW(), total_sent = total_sent + 1, last_level = $1, updated_at = NOW()
		WHERE category = $2`

	result, err := r.db.ExecContext(ctx, query, level, category)
	if err != nil {
		return fmt.Errorf("failed to update sent state for %s: %w", category, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		// Category not in rotation_state yet, create it
		return r.Upsert(ctx, &RotationState{
			Category:  category,
			LastLevel: &level,
		})
	}

	return nil
}

// Reset resets all rotation states (for testing or manual reset).
func (r *RotationRepository) Reset(ctx context.Context) error {
	query := `UPDATE rotation_state SET last_sent_at = NULL, total_sent = 0, last_level = NULL, updated_at = NOW()`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to reset rotation states: %w", err)
	}

	return nil
}

// Delete removes a rotation state entry.
func (r *RotationRepository) Delete(ctx context.Context, category string) error {
	query := `DELETE FROM rotation_state WHERE category = $1`

	result, err := r.db.ExecContext(ctx, query, category)
	if err != nil {
		return fmt.Errorf("failed to delete rotation state for %s: %w", category, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("rotation state for %s not found", category)
	}

	return nil
}
